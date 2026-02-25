package scheduling

import (
	"errors"
	"log"
	"math"
	"strings"

	"github.com/serverledge-faas/serverledge/internal/config"
	"github.com/serverledge-faas/serverledge/internal/function"
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
		if fn.OutputModel.Quality != nil {
			switch strings.ToLower(*fn.OutputModel.Quality) {
			case "high":
				return 0.0
			case "medium":
				return 1.0
			case "low":
				return 2.0
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
	fn       *function.Function
	warm     bool
	energy   float64
	errScore float64
}

// paretoFilter returns only the Pareto-optimal variants under the pair of
// objectives (energy, error), both minimised.
func paretoFilter(candidates []evaluatedVariant) []evaluatedVariant {
	n := len(candidates)
	dominated := make([]bool, n)

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			ci, cj := candidates[i], candidates[j]
			// j weakly dominates i on both axes and strictly on at least one
			if cj.energy <= ci.energy && cj.errScore <= ci.errScore &&
				(cj.energy < ci.energy || cj.errScore < ci.errScore) {
				dominated[i] = true
				break
			}
		}
	}

	var pareto []evaluatedVariant
	for i, v := range candidates {
		if !dominated[i] {
			pareto = append(pareto, v)
		}
	}
	return pareto
}

// ---------------------------------------------------------------------------
// Main entry point
// ---------------------------------------------------------------------------

// SelectParetoVariant selects the best variant using Pareto-scalarisation.
//
// The caller passes QualityWeight λ ∈ [0,1]:
//   - λ = 0 → minimise energy
//   - λ = 1 → minimise error
//   - 0 < λ < 1 → weighted trade-off
//
// Steps:
//  1. Evaluate all registered variants (energy + error).
//  2. Retain only Pareto-optimal variants (non-dominated w.r.t. energy & error).
//  3. Normalise energy and error on the Pareto front.
//  4. Compute score(i) = (1−λ)·E_n(i) + λ·Err_n(i).
//  5. Return the variant with the lowest score.
//
// The full Pareto front (with normalised values and scores) is embedded in the
// returned VariantSchedulingReport so callers can plot energy/error charts.
func SelectParetoVariant(
	r *function.Request,
) (*function.Function, *function.VariantSchedulingReport, error) {

	if r == nil || r.Fun == nil {
		return nil, nil, errors.New("invalid request")
	}

	base := r.Fun
	lambda := *r.QualityWeight // guaranteed non-nil by caller

	report := &function.VariantSchedulingReport{
		LogicalName:     base.LogicalName,
		InvokedFunction: base.Name,
		QualityWeight:   lambda,
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
	// 2. Evaluate every variant
	// ------------------------------------------------------------------
	// NOTE: Pareto comparison uses invocation-only (warm) energy because
	// cold-start cost is a transient runtime effect, not an intrinsic
	// property of the variant's algorithm. Using cold-start would let the
	// runtime platform (native vs python) dominate over algorithmic trade-offs.
	var evaluated []evaluatedVariant
	for _, fn := range variants {
		if fn == nil {
			continue
		}
		warm := node.HasWarmContainer(fn)
		// Always use invocation energy for Pareto ranking; cold-start is
		// accounted for separately in the actual scheduling path.
		invocationEnergy, err := energyCostJoule(fn, true /* warm=true → invocation only */)
		if err != nil {
			continue // skip variants without an energy profile
		}
		evaluated = append(evaluated, evaluatedVariant{
			fn:       fn,
			warm:     warm,
			energy:   invocationEnergy,
			errScore: errorScore(fn),
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

		paretoPoints = append(paretoPoints, function.ParetoPoint{
			FunctionName:  v.fn.Name,
			VariantID:     v.fn.VariantID,
			Energy:        v.energy,
			ErrorEstimate: v.errScore,
			NormEnergy:    normE,
			NormError:     normErr,
			Score:         score,
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

	log.Printf("[pareto] selected=%s  λ=%.2f  score=%.4f  E=%.6f  Err=%.6f",
		report.SelectedFunction, lambda, bestScore, bestV.energy, bestV.errScore)

	return bestV.fn, report, nil
}
