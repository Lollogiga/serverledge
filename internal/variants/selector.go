package variants

import "github.com/serverledge-faas/serverledge/internal/function"

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
