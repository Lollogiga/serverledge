package energy

import (
	"errors"
	"log"
	"time"
)

func Collect(prom *PrometheusClient) error {
	now := time.Now()

	containers := SnapshotContainers()
	//log.Printf("[energy][collect] containers=%d", len(containers))

	for _, state := range containers {

		// 1) energia cumulativa (Kepler)
		jouleNow, err := prom.ReadContainerJoule(state.ContainerID)
		if err != nil {
			if errors.Is(err, ErrNoData) {
				//log.Printf("[energy][collect] container=%s no kepler data yet", state.ContainerID)
				continue
			}
			if errors.Is(err, ErrTransient) {
				//log.Printf("[energy][collect] transient kepler error -> skip cycle")
				return ErrTransient
			}
			//log.Printf("[energy][collect] unexpected error container=%s: %v", state.ContainerID, err)
			return ErrTransient
		}

		// 2) invocazioni cumulative (contatore locale)
		nNow := LoadInvocations(state.ContainerID)

		/*log.Printf(
			"[energy][collect] container=%s E_now=%.6fJ N_now=%d hasBaseline=%v pendingE=%.6f pendingN=%d",
			state.ContainerID, jouleNow, nNow, state.HasValue, state.PendingEnergyJ, state.PendingInvocations,
		)*/

		// 3) baseline init (prima osservazione)
		if !state.HasValue {
			state.LastJoule = jouleNow
			state.LastInvocations = nNow
			state.HasValue = true
			state.LastRead = now

			// reset accumulatori (difensivo)
			state.PendingEnergyJ = 0
			state.PendingInvocations = 0

			log.Printf(
				state.ContainerID, jouleNow, nNow,
			)
			continue
		}

		// 4) delta energia
		dE := jouleNow - state.LastJoule

		// Reset / rinascita container / reset metrica
		if dE < 0 {
			state.LastJoule = jouleNow
			state.LastInvocations = nNow
			state.PendingEnergyJ = 0
			state.PendingInvocations = 0
			state.LastRead = now
			continue
		}

		// Reset invocations (should not happen often, but handle it)
		if nNow < state.LastInvocations {
			state.LastJoule = jouleNow
			state.LastInvocations = nNow
			state.PendingEnergyJ = 0
			state.PendingInvocations = 0
			state.LastRead = now
			continue
		}

		// 5) delta invocazioni
		dN := nNow - state.LastInvocations

		//log.Printf("[energy][collect] container=%s dE=%.6f dN=%d", state.ContainerID, dE, dN)

		// 6) accumula pending (gestione lag Kepler/Prometheus)
		// - dN può arrivare prima di dE
		// - dE può arrivare in un tick successivo con dN=0
		state.PendingEnergyJ += dE
		state.PendingInvocations += dN

		// 7) se ho invocazioni pendenti ma energia non ancora arrivata -> attendo
		if state.PendingInvocations > 0 && state.PendingEnergyJ <= 0 {
			/*log.Printf(
				"[energy][collect] container=%s WAIT energy lag (pendingN=%d pendingE=%.6f)",
				state.ContainerID, state.PendingInvocations, state.PendingEnergyJ,
			)*/
		}

		// 8) quando entrambi sono disponibili, attribuisco energia
		if state.PendingInvocations > 0 && state.PendingEnergyJ > 0 {

			eInv := state.PendingEnergyJ / float64(state.PendingInvocations)

			/*log.Printf(
				"[energy][collect] APPLY container=%s eInv=%.6f (pendingE=%.6f pendingN=%d)",
				state.ContainerID, eInv, state.PendingEnergyJ, state.PendingInvocations,
			)*/

			// (A) aggiorna stima in etcd (stato per scheduler)
			UpdateVariantEnergyEMS(state.FunctionName, eInv)

			log.Printf(
				"[DEBUG][collect-before-write] container=%s fn=%s variant=%s eInv=%.6f",
				state.ContainerID,
				state.FunctionName,
				state.VariantID,
				eInv,
			)

			// (B) salva campione in Influx (storico per grafici)
			writeInvocation(state, eInv)

			// reset accumulatori dopo attribuzione (fondamentale)
			state.PendingEnergyJ = 0
			state.PendingInvocations = 0
		}

		// 9) aggiorna baseline SEMPRE a fine tick
		state.LastJoule = jouleNow
		state.LastInvocations = nNow
		state.LastRead = now
	}

	return nil
}
