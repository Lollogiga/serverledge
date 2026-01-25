package energy

import (
	"context"
	"time"

	"github.com/serverledge-faas/serverledge/internal/influx"
)

func writeColdStart(state *ContainerState, joule float64) {
	writeSample(state, "coldstart_joule", joule)
}

func writeInvocation(state *ContainerState, joule float64) {
	writeSample(state, "invocation_joule", joule)
}

func writeSample(state *ContainerState, field string, joule float64) {
	writer, err := influx.NewWriter()
	if err != nil {
		return
	}
	defer writer.Close()

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

	_ = writer.WritePoint(context.Background(), sample)
}
