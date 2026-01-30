package energy

import (
	"log"

	"github.com/serverledge-faas/serverledge/internal/function"
)

const (
	defaultAlpha = 0.2
)

// UpdateVariantEnergyEMS updates the InvocationJoule of a variant using EMA
func UpdateVariantEnergyEMS(
	variantName string,
	eInv float64,
) {

	fn, exists := function.GetFunction(variantName)
	if !exists || fn == nil {
		log.Printf(
			"[energy][ems] variant %s not found, skipping",
			variantName,
		)
		return
	}

	if fn.EnergyProfile == nil {
		log.Printf(
			"[energy][ems] variant %s has nil EnergyProfile, skipping",
			variantName,
		)
		return
	}

	old := fn.EnergyProfile.InvocationJoule
	alpha := defaultAlpha

	var updated float64
	if old <= 0 {
		// first observation
		updated = eInv
	} else {
		updated = alpha*eInv + (1-alpha)*old
	}

	fn.EnergyProfile.InvocationJoule = updated

	log.Printf(
		"[DEBUG][etcd-update] function=%s variant=%s new_invocation_joule=%.6f",
		fn.Name,
		fn.VariantID,
		eInv,
	)

	if err := fn.SaveToEtcd(); err != nil {
		log.Printf(
			"[energy][ems] failed to update EMS for %s: %v",
			fn.Name,
			err,
		)
		return
	}

	log.Printf(
		"[energy][ems] updated %s: old=%.6f new=%.6f sample=%.6f",
		fn.Name,
		old,
		updated,
		eInv,
	)
}
