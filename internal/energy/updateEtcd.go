package energy

import (
	"log"

	"github.com/serverledge-faas/serverledge/internal/function"
)

func updateInvocationEtcd(state *ContainerState, joule float64) {
	f, ok := function.GetFunction(state.VariantName)
	if !ok {
		log.Printf("[energy] function %s not found (invocation)", state.VariantName)
		return
	}

	if f.EnergyProfile == nil {
		return
	}

	f.EnergyProfile.InvocationJoule = joule

	if err := f.SaveToEtcd(); err != nil {
		log.Printf("[energy] failed to update invocation joule: %v", err)
	}
}

func updateColdStartEtcd(state *ContainerState, joule float64) {
	f, ok := function.GetFunction(state.VariantName)
	if !ok {
		log.Printf("[energy] function %s not found (cold start)", state.VariantName)
		return
	}

	if f.EnergyProfile == nil {
		return
	}

	f.EnergyProfile.ColdStartJoule = joule

	if err := f.SaveToEtcd(); err != nil {
		log.Printf("[energy] failed to update cold start joule: %v", err)
	}
}
