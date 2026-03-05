package scheduling

// carbon_intensity.go — automatic λ derivation from real-time carbon intensity.
//
// Flow (called once per Pareto variant-selection, cached 15 min):
//   1. Read zone and auth-token from the Serverledge config file.
//   2. GET https://api.electricitymaps.com/v3/carbon-intensity/latest?zone=<ZONE>
//   3. Apply a sigmoid normalised on the IEA 2023 world-average (475 gCO2/kWh):
//         ci_norm = (ci - 475) / 475
//         λ       = 1 - 1 / (1 + exp(-ci_norm))
//      Clean grid  (CI ≪ 475) → λ → 1.0  (prioritise accuracy)
//      Dirty grid  (CI ≫ 475) → λ → 0.0  (prioritise energy saving)
//   4. On any error fall back to λ = 0.5 (neutral trade-off).

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/serverledge-faas/serverledge/internal/config"
)

// ciGlobalMidpoint is the IEA World Average carbon intensity for 2023 (gCO2eq/kWh).
const ciGlobalMidpoint = 475.0

// ciCacheTTL is how long a fetched carbon-intensity value is considered fresh.
const ciCacheTTL = 15 * time.Minute

// ciLambdaFallback is the λ value used when the ElectricityMaps API is
// unavailable or not configured.  0.5 = neutral energy/accuracy trade-off.
const ciLambdaFallback = 0.5

// ciCacheEntry holds one cached carbon-intensity value.
type ciCacheEntry struct {
	value     float64
	fetchedAt time.Time
}

// ciCache is a per-zone cache of carbon-intensity values.
var ciCache struct {
	sync.Mutex
	entries map[string]ciCacheEntry // key = zone string ("" = auto-detect)
}

// electricityMapsResponse is the relevant subset of the ElectricityMaps v3 response.
type electricityMapsResponse struct {
	CarbonIntensity float64 `json:"carbonIntensity"`
}

// fetchCarbonIntensity performs a live HTTP call to the ElectricityMaps API.
//
// Configuration (serverledge-conf.yaml):
//
//	electricitymaps:
//	  zone:  "IT-NO"          # ElectricityMaps zone identifier
//	  token: "YOUR_TOKEN"     # API auth token
//
// The token can alternatively be supplied via the ELECTRICITY_MAPS_TOKEN
// environment variable (useful for production deployments where secrets must
// not be committed to config files).
// fetchCarbonIntensityForZone calls the ElectricityMaps API for the given zone.
// If zoneOverride is empty, falls back to the value in the config file.
func fetchCarbonIntensityForZone(zoneOverride string) (float64, error) {
	token := config.GetString(config.ELECTRICITY_MAPS_TOKEN, "")
	if token == "" {
		// env-var fallback
		token = os.Getenv("ELECTRICITY_MAPS_TOKEN")
	}
	if token == "" {
		return 0, fmt.Errorf(
			"electricitymaps: auth token not configured — " +
				"set 'electricitymaps.token' in the config file " +
				"or export ELECTRICITY_MAPS_TOKEN",
		)
	}

	zone := zoneOverride
	if zone == "" {
		zone = config.GetString(config.ELECTRICITY_MAPS_ZONE, "")
	}

	rawURL := "https://api.electricitymaps.com/v3/carbon-intensity/latest"
	if zone != "" {
		rawURL += "?zone=" + zone
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, fmt.Errorf("electricitymaps: could not build request: %w", err)
	}
	req.Header.Set("auth-token", token)

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("electricitymaps: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("electricitymaps: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var payload electricityMapsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, fmt.Errorf("electricitymaps: could not decode response: %w", err)
	}

	return payload.CarbonIntensity, nil
}

// GetCarbonIntensityForZone returns the CI for the given zone (empty = config default),
// honouring a per-zone 15-minute in-memory cache.
func GetCarbonIntensityForZone(zoneOverride string) (float64, error) {
	// Resolve the effective zone key used for caching
	cacheKey := zoneOverride
	if cacheKey == "" {
		cacheKey = config.GetString(config.ELECTRICITY_MAPS_ZONE, "")
	}

	ciCache.Lock()
	defer ciCache.Unlock()

	if ciCache.entries == nil {
		ciCache.entries = make(map[string]ciCacheEntry)
	}

	if e, ok := ciCache.entries[cacheKey]; ok && time.Since(e.fetchedAt) < ciCacheTTL {
		log.Printf("[carbon-intensity] cache hit zone=%q: %.1f gCO2/kWh (age=%s)",
			cacheKey, e.value, time.Since(e.fetchedAt).Round(time.Second))
		return e.value, nil
	}

	ci, err := fetchCarbonIntensityForZone(zoneOverride)
	if err != nil {
		return 0, err
	}

	ciCache.entries[cacheKey] = ciCacheEntry{value: ci, fetchedAt: time.Now()}
	log.Printf("[carbon-intensity] fetched %.1f gCO2/kWh (zone=%q)", ci, cacheKey)
	return ci, nil
}

// CarbonIntensityToLambda converts a carbon-intensity value (gCO2eq/kWh) to a
// Pareto quality-weight λ ∈ (0, 1) via a normalised sigmoid:
//
//	ci_norm = (ci - CI_MID_GLOBAL) / CI_MID_GLOBAL
//	λ       = 1 − 1 / (1 + exp(−ci_norm))
//
// The curve is centred on the IEA 2023 world average (475 gCO2/kWh), which
// maps to λ ≈ 0.5 (balanced trade-off).
//
//   - ci =   0  → λ ≈ 0.731  (very clean grid  → max accuracy)
//   - ci = 475  → λ ≈ 0.500  (world average    → balanced)
//   - ci = 950  → λ ≈ 0.269  (very dirty grid  → max energy saving)
func CarbonIntensityToLambda(ci float64) float64 {
	ciNorm := (ci - ciGlobalMidpoint) / ciGlobalMidpoint
	return 1.0 - 1.0/(1.0+math.Exp(-ciNorm))
}

// GetLambda fetches λ for the configured zone (no override).
func GetLambda() float64 {
	lambda, _ := GetLambdaAndCI("")
	return lambda
}

// GetLambdaAndCI fetches CI for the given zone override (empty = use config)
// and returns both λ and the raw CI value.
// On any error ci=0 and lambda=ciLambdaFallback are returned.
func GetLambdaAndCI(zoneOverride string) (lambda, ci float64) {
	var err error
	ci, err = GetCarbonIntensityForZone(zoneOverride)
	if err != nil {
		log.Printf("[carbon-intensity] WARNING: %v — falling back to λ=%.2f", err, ciLambdaFallback)
		return ciLambdaFallback, 0
	}
	lambda = CarbonIntensityToLambda(ci)
	effZone := zoneOverride
	if effZone == "" {
		effZone = config.GetString(config.ELECTRICITY_MAPS_ZONE, "auto")
	}
	log.Printf("[carbon-intensity] zone=%s CI=%.1f gCO2/kWh → λ=%.4f", effZone, ci, lambda)
	return lambda, ci
}
