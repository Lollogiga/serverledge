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
// for UCB exploration. The LCB formula is:
//
//	LCB = µ - β · σ / √n
//
// where µ, σ, n are derived from historical InfluxDB measurements.
// Being optimistic about energy (assuming it could be lower) drives the
// scheduler to explore under-sampled variants.
//
// Falls back to the etcd energy profile when:
//   - InfluxDB is not configured or unavailable
//   - the variant has fewer than minSamples observations
//   - beta == 0 (UCB disabled)
func energyCostJouleUCB(
	fn *function.Function,
	warm bool,
	beta float64,
	minSamples int,
) (joule float64, usedUCB bool, sampleCount int64, err error) {

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
		// InfluxDB unavailable — silent fallback to etcd value
		if queryErr != influx.ErrInfluxNotConfigured {
			log.Printf("[ucb] influx query error for %s: %v (using etcd fallback)", fn.Name, queryErr)
		}
		return base, false, 0, nil
	}
	if stats.Count < int64(minSamples) {
		log.Printf("[ucb] variant=%s has %d/%d samples — using etcd value (exploration deferred)",
			fn.VariantID, stats.Count, minSamples)
		return base, false, stats.Count, nil
	}

	// LCB: optimistic lower bound for energy minimisation
	lcb := stats.Mean - beta*stats.Stddev/math.Sqrt(float64(stats.Count))
	if lcb < 0 {
		lcb = 0
	}

	log.Printf("[ucb] variant=%s  µ=%.6f  σ=%.6f  n=%d  β=%.2f  LCB=%.6f  etcd=%.6f",
		fn.VariantID, stats.Mean, stats.Stddev, stats.Count, beta, lcb, base)

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

	// λ is derived automatically from the current carbon intensity.
	// r.CIZoneOverride (se non vuoto) sovrascrive la zona da config per questa invocazione.
	lambda, ci := GetLambdaAndCI(r.CIZoneOverride)

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
