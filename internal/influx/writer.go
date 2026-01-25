package influx

import (
	"context"
	"log"
	"os"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Writer struct {
	client influxdb2.Client
	write  api.WriteAPIBlocking // Usa il pacchetto api qui
}

func NewWriter() (*Writer, error) {
	client := newClient() // Assicurati che newClient sia definito correttamente nel tuo package
	if client == nil {
		return nil, ErrInfluxNotConfigured
	}

	org := os.Getenv("INFLUX_ORG")
	bucket := os.Getenv("INFLUX_BUCKET")

	if org == "" || bucket == "" {
		return nil, ErrInfluxNotConfigured
	}

	writeAPI := client.WriteAPIBlocking(org, bucket)

	return &Writer{
		client: client,
		write:  writeAPI,
	}, nil
}

func (w *Writer) Close() {
	w.client.Close()
}

func (w *Writer) WritePoint(ctx context.Context, p Point) error {
	// USA INFLUXDB2.NEWPOINT E LASCIA CHE GO GESTISCA IL TIPO
	point := influxdb2.NewPoint(
		p.Measurement,
		p.Tags,
		p.Fields,
		p.Time,
	)

	log.Printf(
		"[influx] writing point: %s %v %v",
		p.Measurement,
		p.Tags,
		p.Fields,
	)

	// Ora w.write.WritePoint accetterà 'point' senza problemi
	return w.write.WritePoint(ctx, point)
}
