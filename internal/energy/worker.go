package energy

import (
	"log"
	"time"
)

func StartCollector(prom *PrometheusClient, period time.Duration) {
	go func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()

		log.Printf("[energy][worker] started with period=%s", period)

		for range ticker.C {
			log.Printf("[energy][worker] tick")
			if err := Collect(prom); err != nil {
				log.Printf("[energy][worker] tick aborted: %v", err)
			}
		}
	}()
}
