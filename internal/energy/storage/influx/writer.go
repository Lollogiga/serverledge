package influx

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/serverledge-faas/serverledge/internal/energy"
)

type Writer struct {
	client influxdb2.Client
	write  api.WriteAPIBlocking
	bucket string
	org    string
}

// NewWriter crea un writer InfluxDB leggendo da env
func NewWriter() (*Writer, error) {
	url := os.Getenv("INFLUX_URL")
	token := os.Getenv("INFLUX_TOKEN")
	org := os.Getenv("INFLUX_ORG")
	bucket := os.Getenv("INFLUX_BUCKET")

	if url == "" || token == "" || org == "" || bucket == "" {
		return nil, errors.New("missing InfluxDB environment variables")
	}

	client := influxdb2.NewClient(url, token)

	return &Writer{
		client: client,
		write:  client.WriteAPIBlocking(org, bucket),
		bucket: bucket,
		org:    org,
	}, nil
}

// Close chiude il client
func (w *Writer) Close() {
	w.client.Close()
}

// WriteEnergySample scrive UN campione in InfluxDB
func (w *Writer) WriteEnergySample(ctx context.Context, s energy.EnergySample) error {
	if s.Timestamp.IsZero() {
		s.Timestamp = time.Now()
	}

	point := influxdb2.NewPoint(
		"energy_sample",
		map[string]string{
			"logical_name":  s.LogicalName,
			"variant_name":  s.VariantName,
			"runtime":       s.Runtime,
			"is_warm":       boolToString(s.IsWarm),
			"experiment_id": s.ExperimentID,
			"container_id":  s.ContainerID,
		},
		map[string]interface{}{
			"invocation_joule": s.InvocationJoule,
			"duration_ms":      s.DurationMs,
			"init_time_ms":     s.InitTimeMs,
			"queue_time_ms":    s.QueueTimeMs,
		},
		s.Timestamp,
	)

	if s.ColdStartJoule != nil {
		point.AddField("coldstart_joule", *s.ColdStartJoule)
		point.AddField("energy_total_joule", s.InvocationJoule+*s.ColdStartJoule)
	} else {
		point.AddField("energy_total_joule", s.InvocationJoule)
	}

	return w.write.WritePoint(ctx, point)
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
