package energy

import (
	"context"
	"log"
	"time"

	"github.com/serverledge-faas/serverledge/internal/influx"
)

var influxWriter *influx.Writer

func writeColdStart(state *ContainerState, joule float64) {
	writeSample(state, "coldstart_joule", joule)
}

func InitEnergyWriter() error {
	w, err := influx.NewWriter()
	if err != nil {
		return err
	}
	influxWriter = w
	return nil
}

func writeInvocation(state *ContainerState, joule float64) {
	log.Printf(
		"[energy][writer] writeInvocation container=%s joule=%.6f",
		state.ContainerID,
		joule,
	)
	writeSample(state, "invocation_joule", joule)
}

func writeSample(state *ContainerState, field string, joule float64) {
	if influxWriter == nil {
		return
	}

	sample := influx.Point{
		Measurement: "energy_sample",
		Tags: map[string]string{
			"container_id": state.ContainerID,
			"logical_name": state.LogicalName,
			"variant_name": state.VariantName,
			"runtime":      state.Runtime,
		},
		Fields: map[string]interface{}{
			field: joule,
		},
		Time: time.Now(),
	}

	_ = influxWriter.WritePoint(context.Background(), sample)
}
