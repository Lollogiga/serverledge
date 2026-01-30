package energy

import (
	"log"

	"github.com/serverledge-faas/serverledge/internal/function"
)

func updateInvocationEtcd(state *ContainerState, joule float64) {
	f, ok := function.GetFunction(state.FunctionName)
	if !ok {
		log.Printf("[energy] function %s not found (invocation)", state.FunctionName)
		return
	}

	if f.EnergyProfile == nil {
		log.Printf("[energy] function %s has nil EnergyProfile", state.FunctionName)
		return
	}

	f.EnergyProfile.InvocationJoule = joule

	if err := f.SaveToEtcd(); err != nil {
		log.Printf("[energy] failed to update invocation joule for %s: %v",
			state.FunctionName, err)
	}
}

func updateColdStartEtcd(state *ContainerState, joule float64) {
	f, ok := function.GetFunction(state.FunctionName)
	if !ok {
		log.Printf("[energy] function %s not found (cold start)", state.FunctionName)
		return
	}

	if f.EnergyProfile == nil {
		log.Printf("[energy] function %s has nil EnergyProfile", state.FunctionName)
		return
	}

	f.EnergyProfile.ColdStartJoule = joule

	if err := f.SaveToEtcd(); err != nil {
		log.Printf("[energy] failed to update cold start joule for %s: %v",
			state.FunctionName, err)
	}
}
