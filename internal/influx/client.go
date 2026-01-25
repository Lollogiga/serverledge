package influx

import (
	"log"
	"os"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

func newClient() influxdb2.Client {
	url := os.Getenv("INFLUX_URL")
	token := os.Getenv("INFLUX_TOKEN")

	if url == "" || token == "" {
		log.Println("[influx] INFLUX_URL or INFLUX_TOKEN not set")
		return nil
	}

	return influxdb2.NewClient(url, token)
}
