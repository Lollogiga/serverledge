package energy

import (
	"log"
	"time"
)

func Collect(prom *PrometheusClient) {
	for _, state := range SnapshotContainers() {

		joule, err := prom.ReadContainerJoule(state.ContainerID)
		if err != nil {
			// Prometheus non ha ancora dati → NON aggiornare
			continue
		}

		// prima lettura → solo inizializzazione
		if !state.HasValue {
			state.LastJoule = joule
			state.HasValue = true
			state.LastRead = time.Now()
			continue
		}

		delta := joule - state.LastJoule
		if delta <= 0 {
			// contatore non avanzato → ignora
			continue
		}

		// aggiorna stato locale
		state.LastJoule = joule
		state.LastRead = time.Now()

		// scrivi su Influx (RAW SAMPLE)
		writeInvocation(state, delta)

		log.Printf(
			"[energy] container=%s delta_joule=%.6f",
			state.ContainerID,
			delta,
		)
	}
}
