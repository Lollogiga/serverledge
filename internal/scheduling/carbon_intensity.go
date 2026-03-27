package scheduling

// carbon_intensity.go — automatic λ derivation from real-time carbon intensity.
//
// Flow (called once per Pareto variant-selection, cached 15 min):
//   1. Read zone, auth-token, and params-file path from the Serverledge config.
//   2. At first use, load zone_ci_params.csv (path from config key
//      electricitymaps.params_file) to get per-zone sigmoid parameters
//      (ci_mid, s) pre-computed offline by carbon_lambda_analysis.py.
//   3. GET https://api.electricitymaps.com/v3/carbon-intensity/latest?zone=<ZONE>
//      for the current CI value (cached 15 min).
//   4. Compute λ via a sigmoid centred on the zone's own historical mean:
//
//         λ = 1 / (1 + exp(K * (CI - ci_mid) / s))
//
//      where K=0.45, ci_mid = historical mean, s = (P95-P5)/(2·ln((1-α)/α)).
//      This makes "high" and "low" relative to the region's own baseline.
//      Clean grid  (CI ≪ ci_mid) → λ → 1.0  (prioritise accuracy)
//      Dirty grid  (CI ≫ ci_mid) → λ → 0.0  (prioritise energy saving)
//   5. On any error fall back to λ = 0.5 (neutral trade-off).

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/serverledge-faas/serverledge/internal/config"
)

// ciMedFallback is the global fallback midpoint (gCO2eq/kWh) used when no
// zone-specific historical data is available.
const ciMedFallback = 425.0

// ciSFallback is the global fallback scale factor used together with ciMedFallback.
const ciSFallback = 68.4

// ciK is the global fallback shape parameter of the sigmoid (used when no
// per-zone k is present in the params CSV).
const ciK = 0.45

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

// CIZoneParams holds the pre-computed sigmoid parameters for a specific zone,
// derived offline from historical data via carbon_lambda_analysis.py.
type CIZoneParams struct {
	CIMid float64 // historical mean gCO2eq/kWh (sigmoid centre, λ=0.5 here)
	S     float64 // scale factor  s = (P95-P5)/(2·ln((1-α)/α))
	K     float64 // sigmoid steepness (default ciK=0.45 if not specified in CSV)
}

// zoneParamsOnce ensures the zone params CSV is loaded at most once.
var zoneParamsOnce sync.Once

// zoneParamsMap holds the loaded per-zone sigmoid parameters.
var zoneParamsMap map[string]CIZoneParams

// loadZoneParamsCSV reads a CSV file with columns [zone, ci_mid, s] and
// returns a map of pre-computed sigmoid parameters per zone.
// Lines starting with '#' and the header row are skipped automatically.
func loadZoneParamsCSV(path string) (map[string]CIZoneParams, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("zone-params: cannot open %q: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comment = '#'
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("zone-params: CSV parse error in %q: %w", path, err)
	}

	m := make(map[string]CIZoneParams, len(records))
	for i, rec := range records {
		if len(rec) < 3 {
			continue
		}
		zone := strings.TrimSpace(rec[0])
		if strings.EqualFold(zone, "zone") {
			continue // header row
		}
		ciMid, err1 := strconv.ParseFloat(strings.TrimSpace(rec[1]), 64)
		s, err2 := strconv.ParseFloat(strings.TrimSpace(rec[2]), 64)
		if err1 != nil || err2 != nil {
			log.Printf("[carbon-intensity] zone-params: skipping row %d (parse error)", i+1)
			continue
		}
		// k is optional (4th column); falls back to the global ciK constant.
		k := ciK
		if len(rec) >= 4 {
			if kv, err := strconv.ParseFloat(strings.TrimSpace(rec[3]), 64); err == nil && kv > 0 {
				k = kv
			}
		}
		m[zone] = CIZoneParams{CIMid: ciMid, S: s, K: k}
	}
	return m, nil
}

// getZoneParams returns the pre-loaded sigmoid parameters for a zone.
// The CSV is loaded lazily on first call from the path in the config file
// (key: electricitymaps.params_file, default: "zone_ci_params.csv").
func getZoneParams(zone string) (CIZoneParams, bool) {
	zoneParamsOnce.Do(func() {
		path := config.GetString(config.ELECTRICITY_MAPS_PARAMS_FILE, "zone_ci_params.csv")
		m, err := loadZoneParamsCSV(path)
		if err != nil {
			log.Printf("[carbon-intensity] WARNING: %v — zone-specific sigmoid params unavailable", err)
			zoneParamsMap = make(map[string]CIZoneParams)
		} else {
			zoneParamsMap = m
			log.Printf("[carbon-intensity] loaded sigmoid params for %d zone(s) from %q", len(m), path)
		}
	})
	p, ok := zoneParamsMap[zone]
	return p, ok
}

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

// CarbonIntensityToLambdaWithStats converts a carbon-intensity value to
// λ ∈ (0, 1) using pre-loaded zone-specific parameters:
//
//	λ = 1 / (1 + exp(k * (CI - ciMid) / s))
//
// ciMid is the zone's historical mean (λ=0.5 here).
// s is the scale factor derived from the zone's P5/P95 range.
// k controls the steepness of the sigmoid (per-zone, from the params CSV).
//   - ci ≪ ciMid → λ → 1.0  (cleaner than usual  → prioritise accuracy)
//   - ci = ciMid → λ = 0.500 (average conditions  → balanced)
//   - ci ≫ ciMid → λ → 0.0  (dirtier than usual  → prioritise energy saving)
func CarbonIntensityToLambdaWithStats(ci, ciMid, s, k float64) float64 {
	if s < 1.0 {
		s = 1.0 // guard against zero/near-zero scale
	}
	if k <= 0 {
		k = ciK
	}
	return 1.0 / (1.0 + math.Exp(k*(ci-ciMid)/s))
}

// CarbonIntensityToLambda converts a carbon-intensity value (gCO2eq/kWh) to a
// Pareto quality-weight λ ∈ (0, 1) using the global fallback baseline.
// Prefer CarbonIntensityToLambdaWithStats when zone-specific params are available.
func CarbonIntensityToLambda(ci float64) float64 {
	return CarbonIntensityToLambdaWithStats(ci, ciMedFallback, ciSFallback, ciK)
}

// GetLambda fetches λ for the configured zone (no override).
func GetLambda() float64 {
	lambda, _ := GetLambdaAndCI("")
	return lambda
}

// GetLambdaAndCI fetches CI for the given zone override (empty = use config)
// and returns both λ and the raw CI value.
// The sigmoid is centred on the zone's own pre-computed ci_mid and s loaded
// from zone_ci_params.csv at startup, so "high" and "low" are always
// relative to that region's baseline.
// On any error ci=0 and lambda=ciLambdaFallback are returned.
func GetLambdaAndCI(zoneOverride string) (lambda, ci float64) {
	var err error
	ci, err = GetCarbonIntensityForZone(zoneOverride)
	if err != nil {
		log.Printf("[carbon-intensity] WARNING: %v — falling back to λ=%.2f", err, ciLambdaFallback)
		return ciLambdaFallback, 0
	}

	effZone := zoneOverride
	if effZone == "" {
		effZone = config.GetString(config.ELECTRICITY_MAPS_ZONE, "")
	}

	if effZone != "" {
		if params, ok := getZoneParams(effZone); ok {
			lambda = CarbonIntensityToLambdaWithStats(ci, params.CIMid, params.S, params.K)
			log.Printf("[carbon-intensity] zone=%s CI=%.1f gCO2/kWh (ci_mid=%.1f s=%.2f k=%.2f) → λ=%.4f",
				effZone, ci, params.CIMid, params.S, params.K, lambda)
			return lambda, ci
		}
		log.Printf("[carbon-intensity] WARNING: no zone params for %q — using global baseline", effZone)
	}

	lambda = CarbonIntensityToLambda(ci)
	log.Printf("[carbon-intensity] zone=%s CI=%.1f gCO2/kWh → λ=%.4f (global baseline)",
		effZone, ci, lambda)
	return lambda, ci
}
