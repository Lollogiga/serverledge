package scheduling

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/serverledge-faas/serverledge/internal/container"
	"github.com/serverledge-faas/serverledge/internal/energy"
	"github.com/serverledge-faas/serverledge/internal/energy/collector"
	"github.com/serverledge-faas/serverledge/internal/energy/storage/influx"
	"github.com/serverledge-faas/serverledge/internal/executor"
)

const HANDLER_DIR = "/app"

// Execute serves a request on the specified container.
func Execute(cont *container.Container, r *scheduledRequest, isWarm bool) error {

	log.Printf("[%s] Executing on container: %v", r.Fun, cont.ID)

	var req executor.InvocationRequest
	if r.Fun.Runtime == container.CUSTOM_RUNTIME {
		req = executor.InvocationRequest{
			Params:       r.Params,
			ReturnOutput: r.ReturnOutput,
		}
	} else {
		cmd := container.RuntimeToInfo[r.Fun.Runtime].InvocationCmd
		req = executor.InvocationRequest{
			Command:      cmd,
			Params:       r.Params,
			Handler:      r.Fun.Handler,
			HandlerDir:   HANDLER_DIR,
			ReturnOutput: r.ReturnOutput,
		}
	}

	// =========================
	// Function execution
	// =========================
	t0 := time.Now()
	initTime := t0.Sub(r.Arrival).Seconds()

	response, invocationWait, err := container.Execute(cont.ID, &req)
	if err != nil {
		logs, errLog := container.GetLog(cont.ID)
		if errLog == nil {
			fmt.Println(logs)
		} else {
			fmt.Printf("Failed to get log: %v\n", errLog)
		}

		completions <- &completionNotification{r: r, cont: cont, failed: true}
		return fmt.Errorf("[%s] Execution failed on container %v: %v ", r, cont.ID, err)
	}

	if !response.Success {
		completions <- &completionNotification{r: r, cont: cont, failed: true}
		return fmt.Errorf("[%s] Function execution failed %v", r, cont.ID)
	}

	// =========================
	// Update execution report
	// =========================
	r.Result = response.Result
	r.Output = response.Output
	r.IsWarmStart = isWarm
	r.Duration = time.Since(t0).Seconds() - invocationWait.Seconds()
	r.ResponseTime = time.Since(r.Arrival).Seconds()
	r.InitTime = initTime + invocationWait.Seconds()

	// =========================
	// Read real energy from Kepler (via Prometheus)
	// =========================
	prom := collector.NewPrometheusCollector("http://localhost:9090")

	invocationJoule, err := prom.MeasureInvocationJoule(
		context.Background(),
		string(cont.ID),
	)

	if err != nil {
		log.Printf("[WARN] Kepler read failed (invocation): %v", err)
		invocationJoule = 0
	}

	// =========================
	// Persist raw energy sample to InfluxDB
	// =========================
	writer, werr := influx.NewWriter()
	if werr != nil {
		log.Printf("[WARN] Influx writer not available: %v", werr)
	} else {
		defer writer.Close()

		sample := energy.EnergySample{
			Timestamp:    time.Now(),
			LogicalName:  r.Fun.LogicalName,
			VariantName:  r.Fun.Name,
			Runtime:      r.Fun.Runtime,
			IsWarm:       isWarm,
			ExperimentID: os.Getenv("INFLUX_EXPERIMENT_ID"),

			InvocationJoule: invocationJoule,
			ColdStartJoule:  nil, // per ora NON separato

			DurationMs:  int64(r.Duration * 1000),
			InitTimeMs:  int64(r.InitTime * 1000),
			QueueTimeMs: 0,

			ContainerID: string(cont.ID),
		}

		if err := writer.WriteEnergySample(context.Background(), sample); err != nil {
			log.Printf("[WARN] Failed to write energy sample: %v", err)
		}
	}

	// =========================
	// Notify scheduler
	// =========================
	completions <- &completionNotification{r: r, cont: cont, failed: false}

	return nil
}
