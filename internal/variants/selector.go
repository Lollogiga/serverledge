package variants

import (
	"errors"
	"math"
	"strings"

	"github.com/serverledge-faas/serverledge/internal/function"
)

const energyEpsilonRatio = 0.05 // 5%

// outputScore returns a comparable score for a variant output.
// Lower is better.
func outputScore(v function.Variant) (float64, error) {
	switch strings.ToLower(v.Output.Type) {

	case "error":
		if v.Output.ErrorEstimate == nil {
			return math.Inf(1), errors.New("missing error_estimate")
		}
		return *v.Output.ErrorEstimate, nil

	case "quality":
		if v.Output.Quality == nil {
			return math.Inf(1), errors.New("missing quality")
		}

		switch strings.ToLower(*v.Output.Quality) {
		case "high":
			return 0.0, nil
		case "medium":
			return 1.0, nil
		case "low":
			return 2.0, nil
		default:
			return math.Inf(1), errors.New("unknown quality value")
		}

	default:
		return math.Inf(1), errors.New("unknown output type")
	}
}

func energyCost(v function.Variant, isWarm bool) float64 {
	if isWarm {
		return v.Energy.InvocationJoule
	}
	return v.Energy.ColdStartJoule
}

func energyEpsilon(maxEnergy float64) float64 {
	return maxEnergy * energyEpsilonRatio
}

// SelectVariantJouleGuard selects the best variant according to
// energy-aware constraints. If no suitable variant is found,
// the original function is returned.
//
// NOTE: This is a stub implementation that preserves legacy behavior.
func SelectVariantJouleGuard(
	fn *function.Function,
	allowApprox bool,
	maxEnergy *float64,
	isWarm bool,
) (*function.Function, error) {

	// Legacy behavior: no variant selection
	return fn, nil
}
