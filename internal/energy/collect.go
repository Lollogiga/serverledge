package energy

import (
	"errors"
	"log"
	"time"
)

func Collect(prom *PrometheusClient) error {
	now := time.Now()

	containers := SnapshotContainers()
	log.Printf("[energy][collect] containers=%d", len(containers))

	for _, state := range containers {

		// 1) energia cumulativa
		jouleNow, err := prom.ReadContainerJoule(state.ContainerID)
		if err != nil {
			if errors.Is(err, ErrNoData) {
				log.Printf("[energy][collect] container=%s no kepler data yet", state.ContainerID)
				continue
			}
			if errors.Is(err, ErrTransient) {
				log.Printf("[energy][collect] transient kepler error -> skip cycle")
				return ErrTransient
			}
			log.Printf("[energy][collect] unexpected error container=%s: %v", state.ContainerID, err)
			return ErrTransient
		}

		// 2) invocazioni cumulative
		nNow := LoadInvocations(state.ContainerID)

		log.Printf(
			"[energy][collect] container=%s E_now=%.6fJ N_now=%d hasBaseline=%v pendingE=%.6f pendingN=%d",
			state.ContainerID, jouleNow, nNow, state.HasValue, state.PendingEnergyJ, state.PendingInvocations,
		)

		// 3) baseline init
		if !state.HasValue {
			state.LastJoule = jouleNow
			state.LastInvocations = nNow // qui ora va bene: baseline parte dall’osservato
			state.HasValue = true
			state.HasInvValue = true
			state.LastRead = now

			log.Printf(
				"[energy][collect] BASELINE INIT container=%s E=%.6f N_base=%d",
				state.ContainerID, jouleNow, nNow,
			)
			continue
		}

		// 4) delta
		dE := jouleNow - state.LastJoule

		// gestisci possibili reset (container rinato / metrica ripartita)
		if dE < 0 {
			log.Printf("[energy][collect] RESET detected container=%s (dE<0). Re-baselining.", state.ContainerID)
			state.LastJoule = jouleNow
			state.LastInvocations = nNow
			state.PendingEnergyJ = 0
			state.PendingInvocations = 0
			state.LastRead = now
			continue
		}

		if nNow < state.LastInvocations {
			log.Printf("[energy][collect] RESET detected container=%s (N decreased). Re-baselining.", state.ContainerID)
			state.LastJoule = jouleNow
			state.LastInvocations = nNow
			state.PendingEnergyJ = 0
			state.PendingInvocations = 0
			state.LastRead = now
			continue
		}

		dN := nNow - state.LastInvocations

		log.Printf("[energy][collect] container=%s dE=%.6f dN=%d", state.ContainerID, dE, dN)

		// 5) accumula in pending
		// - se Kepler “lagga”, dE può arrivare dopo dN
		state.PendingEnergyJ += dE
		state.PendingInvocations += dN

		// 6) se ho invocazioni pendenti ma energia ancora 0 → aspetto tick successivo
		if state.PendingInvocations == 0 {
			// nulla da attribuire
		} else if state.PendingEnergyJ <= 0 {
			log.Printf(
				"[energy][collect] container=%s WAIT energy lag (pendingN=%d pendingE=%.6f)",
				state.ContainerID, state.PendingInvocations, state.PendingEnergyJ,
			)
		} else {
			// 7) finalmente attribuisco energia alle invocazioni accumulate
			eInv := state.PendingEnergyJ / float64(state.PendingInvocations)

			log.Printf(
				"[energy][collect] WILL WRITE container=%s eInv=%.6f (pendingE=%.6f pendingN=%d)",
				state.ContainerID, eInv, state.PendingEnergyJ, state.PendingInvocations,
			)

			writeInvocation(state, eInv)

			// NB: EMA/EMS in etcd -> step 3
			// updateInvocationEtcd(state, eInv)

			// reset pending
			state.PendingEnergyJ = 0
			state.PendingInvocations = 0
		}

		// 8) aggiorna baseline SEMPRE a fine tick
		state.LastJoule = jouleNow
		state.LastInvocations = nNow
		state.LastRead = now
	}

	return nil
}
