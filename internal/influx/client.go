package influx

import (
	"log"
	"os"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/serverledge-faas/serverledge/internal/config"
)

// resolveInfluxParam returns the config-file value if set, otherwise the env var fallback.
func resolveInfluxParam(configKey, envKey string) string {
	v := config.GetString(configKey, "")
	if v != "" {
		return v
	}
	return os.Getenv(envKey)
}

func newClient() influxdb2.Client {
	url := resolveInfluxParam(config.InfluxURL, "INFLUX_URL")
	token := resolveInfluxParam(config.InfluxToken, "INFLUX_TOKEN")

	if url == "" || token == "" {
		log.Println("[influx] INFLUX_URL or INFLUX_TOKEN not set (check config file or env vars)")
		return nil
	}

	return influxdb2.NewClient(url, token)
}
