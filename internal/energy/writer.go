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
	if influxWriter == nil {
		log.Printf("[DEBUG][influx] writer is NIL → skipping write")
		return
	}

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
			"container_id":  state.ContainerID,
			"logical_name":  state.LogicalName,
			"function_name": state.FunctionName,
			"variant_id":    state.VariantID,
		},
		Fields: map[string]interface{}{
			field: joule,
		},
		Time: time.Now(),
	}

	log.Printf(
		"[DEBUG][influx-write] measurement=%s tags=%v field=%s value=%.6f",
		sample.Measurement,
		sample.Tags,
		field,
		joule,
	)

	_ = influxWriter.WritePoint(context.Background(), sample)
}
