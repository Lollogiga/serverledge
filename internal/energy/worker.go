package energy

import "time"

func StartCollector(prom *PrometheusClient, period time.Duration) {
	go func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()

		for range ticker.C {
			Collect(prom)
		}
	}()
}
