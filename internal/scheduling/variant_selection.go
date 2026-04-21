package scheduling

import (
	"context"
	"errors"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/serverledge-faas/serverledge/internal/config"
	"github.com/serverledge-faas/serverledge/internal/function"
	"github.com/serverledge-faas/serverledge/internal/influx"
	"github.com/serverledge-faas/serverledge/internal/node"
)

const worstErrorScore = 1e12

// ---------------------------------------------------------------------------
// Metric helpers
// ---------------------------------------------------------------------------

func energyCostJoule(fn *function.Function, warm bool) (float64, error) {
	if fn.EnergyProfile == nil {
		return worstErrorScore, errors.New("missing energy profile")
	}

	includeColdStart := config.GetBool(config.SchedulingEnergyIncludeColdStart, true)

	if warm {
		return fn.EnergyProfile.InvocationJoule, nil
	}
	if includeColdStart {
		return fn.EnergyProfile.ColdStartJoule + fn.EnergyProfile.InvocationJoule, nil
	}
	return fn.EnergyProfile.InvocationJoule, nil
}

func errorScore(fn *function.Function) float64 {
	if fn.OutputModel == nil {
		return 0.0 // exact by convention
	}
	switch fn.OutputModel.Type {
	case "error":
		if fn.OutputModel.ErrorEstimate != nil {
			return *fn.OutputModel.ErrorEstimate
		}
		return worstErrorScore
	case "quality":
		// Prefer numeric quality_score when available: error = 1 - accuracy.
		// This gives a precise, continuous metric instead of a coarse ordinal label.
		if fn.OutputModel.QualityScore != nil {
			s := *fn.OutputModel.QualityScore
			if s < 0 {
				s = 0
			}
			if s > 1 {
				s = 1
			}
			return 1.0 - s
		}
		// Legacy ordinal fallback ("high"→0, "medium"→0.5, "low"→1).
		if fn.OutputModel.Quality != nil {
			switch strings.ToLower(*fn.OutputModel.Quality) {
			case "high":
				return 0.0
			case "medium":
				return 0.5
			case "low":
				return 1.0
			}
		}
		return worstErrorScore
	}
	return worstErrorScore
}

// ---------------------------------------------------------------------------
// Pareto helpers
// ---------------------------------------------------------------------------

type evaluatedVariant struct {
	fn         *function.Function
	warm       bool
	energy     float64
	errScore   float64
	ucbUsed    bool  // true if LCB adjusted the energy estimate
	ucbSamples int64 // number of InfluxDB samples used
}

// energyCostJouleUCB returns the Lower Confidence Bound (LCB) energy estimate
// for UCB exploration. The formula combines EMA and InfluxDB statistics:
//
//	LCB = µ - β · σ / √n_eff
//
// where:
//   - µ       = stats.Mean (real InfluxDB average) when n > 0,
//               otherwise the EMA prior stored in etcd.
//   - σ       = stats.Stddev when n ≥ minSamples (statistically reliable);
//               otherwise priorSigmaFraction · µ_EMA (conservative prior).
//   - n_eff   = n + priorVirtualCount (always > 0).
//               n=0 → n_eff=0.5 → largest bonus (most exploration).
//               n→∞ → bonus → 0 (well-known variant, no correction needed).
//
// Exploration is ALWAYS active: a variant with 0 samples receives the
// largest possible LCB correction, never falls back to EMA only.
//
// Falls back to the EMA alone (no LCB correction) only when:
//   - InfluxDB is not configured or unavailable.
//   - beta == 0 (exploration disabled).

// priorSigmaFraction is the fraction of µ_EMA used as σ when real statistics
// are not yet reliable (n < minSamples). 0.30 encodes ±30 % prior uncertainty.
const priorSigmaFraction = 0.30

// priorVirtualCount prevents division-by-zero and ensures n=0 always has a
// strictly larger correction than n=1.
const priorVirtualCount = 0.5

func energyCostJouleUCB(
	fn *function.Function,
	warm bool,
	beta float64,
	minSamples int,
) (joule float64, usedUCB bool, sampleCount int64, err error) {

	// base == µ_EMA: the offline/EMA prior persisted in etcd.
	base, baseErr := energyCostJoule(fn, warm)
	if baseErr != nil {
		return 0, false, 0, baseErr
	}
	if fn.VariantID == "" || beta <= 0 {
		return base, false, 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// fn.Name matches the function_name tag in InfluxDB (e.g. "montecarlo-n1000")
	stats, queryErr := influx.QueryEnergyStats(ctx, fn.Name, "24h")
	if queryErr != nil {
		// InfluxDB unavailable — use EMA alone, no exploration correction.
		if queryErr != influx.ErrInfluxNotConfigured {
			log.Printf("[ucb] influx query error for %s: %v (using EMA only)", fn.Name, queryErr)
		}
		return base, false, 0, nil
	}

	// (1) Best available mean: real InfluxDB data when present, EMA prior otherwise.
	mu := base
	if stats.Count > 0 {
		mu = stats.Mean
	}

	// (2) Uncertainty estimate: real σ when statistically reliable,
	//     a prior fraction of µ_EMA when data are scarce (including n=0).
	sigma := priorSigmaFraction * base
	if stats.Count >= int64(minSamples) {
		sigma = stats.Stddev
	}

	// (3) Effective sample count: always > 0, monotonically encodes confidence.
	//     n=0 → nEff=0.5 (biggest bonus); n=5 → nEff=5.5 (smaller bonus).
	nEff := float64(stats.Count) + priorVirtualCount

	// (4) LCB = µ - β · σ / √n_eff  (clamped to 0 — energy cannot be negative).
	lcb := mu - beta*sigma/math.Sqrt(nEff)
	if lcb < 0 {
		lcb = 0
	}

	log.Printf("[ucb] variant=%s  n=%d  µ=%.6f  σ=%.6f  nEff=%.1f  β=%.2f  LCB=%.6f",
		fn.VariantID, stats.Count, mu, sigma, nEff, beta, lcb)

	return lcb, true, stats.Count, nil
}

// paretoFilter returns only the Pareto-optimal variants under the pair of
// objectives (energy, error), both minimised.
//
// Algorithm: O(n log n) sort + single-pass scan (optimal for 2 objectives).
//
//  1. Sort candidates by energy ascending; ties broken by errScore ascending
//     so that among equal-energy points the one with lower error dominates.
//  2. Walk left to right tracking errMin = minimum errScore on the front so far.
//     A point is Pareto-optimal iff its errScore is STRICTLY less than errMin:
//     all previously seen points have energy ≤ current, so the only escape
//     from domination is a strictly lower error.
func paretoFilter(candidates []evaluatedVariant) []evaluatedVariant {
	if len(candidates) == 0 {
		return nil
	}

	// Step 1 – sort
	sorted := make([]evaluatedVariant, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].energy != sorted[j].energy {
			return sorted[i].energy < sorted[j].energy
		}
		return sorted[i].errScore < sorted[j].errScore
	})

	// Step 2 – single scan
	var pareto []evaluatedVariant
	errMin := math.MaxFloat64
	for _, v := range sorted {
		if v.errScore < errMin {
			pareto = append(pareto, v)
			errMin = v.errScore
		}
	}
	return pareto
}

// ---------------------------------------------------------------------------
// Main entry point
// ---------------------------------------------------------------------------

// SelectParetoVariant selects the best variant using Pareto-scalarisation.
//
// λ ∈ (0,1) is computed automatically from the real-time grid carbon intensity
// (ElectricityMaps API, cached 15 min):
//   - clean grid (low CI)  → λ → 1  (prioritise accuracy)
//   - dirty grid (high CI) → λ → 0  (prioritise energy saving)
//
// Steps:
//  1. Fetch λ and raw CI from ElectricityMaps (or use cached value).
//  2. Evaluate all registered variants (energy + error).
//  3. Retain only Pareto-optimal variants (non-dominated w.r.t. energy & error).
//  4. Normalise energy and error on the Pareto front.
//  5. Compute score(i) = (1−λ)·E_n(i) + λ·Err_n(i).
//  6. Return the variant with the lowest score.
//
// The full Pareto front plus λ and CI are embedded in the returned
// VariantSchedulingReport and surfaced in the invocation JSON response.
func SelectParetoVariant(
	r *function.Request,
) (*function.Function, *function.VariantSchedulingReport, error) {

	if r == nil || r.Fun == nil {
		return nil, nil, errors.New("invalid request")
	}

	base := r.Fun

	// ---------------------------------------------------------------------------
	// λ resolution — priority order:
	//   1. r.LambdaOverride != nil  → use directly (experiment fixed-lambda modes)
	//   2. r.CIOverride > 0         → skip API call, convert CI value to λ via sigmoid
	//   3. r.CIZoneOverride != ""   → override zone, then fetch CI from ElectricityMaps
	//   4. default                  → use zone from config, fetch CI from ElectricityMaps
	// ---------------------------------------------------------------------------
	var lambda, ci float64

	switch {
	case r.LambdaOverride != nil:
		// Fixed-lambda baseline mode: bypass carbon intensity entirely.
		lambda = *r.LambdaOverride
		ci = 0 // CI is not meaningful/available in this mode
		log.Printf("[pareto] lambda_override=%.4f — skipping CI lookup", lambda)

	case r.CIOverride > 0:
		// CSV-based experiment mode: CI value supplied directly by the client;
		// convert to λ using the zone's pre-loaded sigmoid parameters.
		ci = r.CIOverride
		effZone := r.CIZoneOverride
		if effZone == "" {
			effZone = config.GetString(config.ELECTRICITY_MAPS_ZONE, "")
		}
		if effZone != "" {
			if params, ok := getZoneParams(effZone); ok {
				lambda = CarbonIntensityToLambdaWithStats(ci, params.CIMid, params.S, params.K)
				log.Printf("[pareto] ci_override=%.1f zone=%s → λ=%.4f (zone sigmoid)", ci, effZone, lambda)
				break
			}
		}
		lambda = CarbonIntensityToLambda(ci)
		log.Printf("[pareto] ci_override=%.1f zone=%q → λ=%.4f (global sigmoid fallback)", ci, effZone, lambda)

	default:
		// Normal mode: fetch real-time CI from ElectricityMaps (cached 15 min).
		lambda, ci = GetLambdaAndCI(r.CIZoneOverride)
	}

	report := &function.VariantSchedulingReport{
		LogicalName:         base.LogicalName,
		InvokedFunction:     base.Name,
		QualityWeight:       lambda,
		CarbonIntensityGCO2: ci,
	}

	// ------------------------------------------------------------------
	// 1. Load all variants registered under the same logical name
	// ------------------------------------------------------------------
	variants, err := function.GetFunctionsByLogicalName(base.LogicalName)
	if err != nil || len(variants) == 0 {
		report.DecisionReason = "fallback-base"
		return base, report, nil
	}

	log.Printf("[pareto] logical=%s found %d candidates", base.LogicalName, len(variants))

	// ------------------------------------------------------------------
	// 2. Evaluate every variant (with optional UCB energy estimate)
	// ------------------------------------------------------------------
	// Energy cost used for Pareto ranking reflects the *actual* cost the
	// scheduler would pay at this moment:
	//   - warm container available → invocation_joule only
	//   - no warm container       → cold_start_joule + invocation_joule
	//
	// When UCB is enabled (beta > 0), the energy is replaced by the Lower
	// Confidence Bound derived from InfluxDB history:
	//   LCB = µ - β·σ/√n
	// This promotes exploration of under-sampled variants (optimism under
	// uncertainty), embodying the UCB acquisition function principle.
	ucbBeta := config.GetFloat(config.SchedulingUCBBeta, 1.0)
	if r.BetaOverride != nil {
		ucbBeta = *r.BetaOverride
	}
	ucbMinSamples := config.GetInt(config.SchedulingUCBMinSamples, 5)

	var evaluated []evaluatedVariant
	for _, fn := range variants {
		if fn == nil {
			continue
		}
		warm := node.HasWarmContainer(fn)
		effectiveEnergy, ucbUsed, ucbSamples, err := energyCostJouleUCB(fn, warm, ucbBeta, ucbMinSamples)
		if err != nil {
			continue // skip variants without an energy profile
		}
		evaluated = append(evaluated, evaluatedVariant{
			fn:         fn,
			warm:       warm,
			energy:     effectiveEnergy,
			errScore:   errorScore(fn),
			ucbUsed:    ucbUsed,
			ucbSamples: ucbSamples,
		})
	}

	if len(evaluated) == 0 {
		report.DecisionReason = "fallback-base"
		return base, report, nil
	}

	// ------------------------------------------------------------------
	// 3. Pareto filtering
	// ------------------------------------------------------------------
	pareto := paretoFilter(evaluated)
	if len(pareto) == 0 {
		// Defensive: should not happen, but fall back gracefully
		pareto = evaluated
	}

	log.Printf("[pareto] front: %d/%d variants non-dominated", len(pareto), len(evaluated))

	// ------------------------------------------------------------------
	// 4. Compute min/max for normalisation over the Pareto front
	// ------------------------------------------------------------------
	eMin, eMax := pareto[0].energy, pareto[0].energy
	errMin, errMax := pareto[0].errScore, pareto[0].errScore
	for _, v := range pareto[1:] {
		if v.energy < eMin {
			eMin = v.energy
		}
		if v.energy > eMax {
			eMax = v.energy
		}
		if v.errScore < errMin {
			errMin = v.errScore
		}
		if v.errScore > errMax {
			errMax = v.errScore
		}
	}

	eRange := eMax - eMin
	errRange := errMax - errMin

	// ------------------------------------------------------------------
	// 5. Score each Pareto-optimal variant and record Pareto front data
	// ------------------------------------------------------------------
	var bestV evaluatedVariant
	bestScore := math.MaxFloat64

	var ucbAnyActive bool
	paretoPoints := make([]function.ParetoPoint, 0, len(pareto))
	for _, v := range pareto {
		var normE, normErr float64
		if eRange > 0 {
			normE = (v.energy - eMin) / eRange
		}
		if errRange > 0 {
			normErr = (v.errScore - errMin) / errRange
		}

		score := (1-lambda)*normE + lambda*normErr

		if v.ucbUsed {
			ucbAnyActive = true
		}

		paretoPoints = append(paretoPoints, function.ParetoPoint{
			FunctionName:   v.fn.Name,
			VariantID:      v.fn.VariantID,
			Energy:         v.energy,
			ErrorEstimate:  v.errScore,
			NormEnergy:     normE,
			NormError:      normErr,
			Score:          score,
			UCBExploration: v.ucbUsed,
			UCBSampleCount: v.ucbSamples,
		})

		if score < bestScore {
			bestScore = score
			bestV = v
		}
	}

	// ------------------------------------------------------------------
	// 6. Build report and return
	// ------------------------------------------------------------------
	report.SelectedFunction = bestV.fn.Name
	report.VariantID = bestV.fn.VariantID
	report.EstimatedEnergy = bestV.energy
	report.WarmHint = bestV.warm
	report.ErrorEstimate = bestV.errScore
	report.DecisionReason = "pareto-scalarisation"
	report.ParetoFront = paretoPoints
	report.UCBActive = ucbAnyActive

	log.Printf("[pareto] selected=%s  λ=%.2f  score=%.4f  E=%.6f  Err=%.6f",
		report.SelectedFunction, lambda, bestScore, bestV.energy, bestV.errScore)

	return bestV.fn, report, nil
}
