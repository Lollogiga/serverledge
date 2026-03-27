package influx

import (
	"context"
	"fmt"
	"math"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/serverledge-faas/serverledge/internal/config"
)

// EnergyStats summarises historical energy measurements for one variant,
// derived from the "energy_sample" InfluxDB measurement.
type EnergyStats struct {
	Mean   float64 // mean invocation_joule
	Stddev float64 // population standard deviation
	Count  int64   // number of observations
}

// QueryEnergyStats fetches mean, stddev and count of invocation_joule for the
// given functionName (= function_name tag, e.g. "montecarlo-n1000") over the
// supplied Flux duration (e.g. "24h", "7d").
// functionName matches the function_name tag written by energy/writer.go.
//
// Returns ErrInfluxNotConfigured when the required environment variables
// (INFLUX_URL, INFLUX_TOKEN, INFLUX_ORG, INFLUX_BUCKET) are not set.
// Returns an empty EnergyStats (Count==0) when no data exist for the variant.
func QueryEnergyStats(ctx context.Context, functionName, window string) (EnergyStats, error) {
	url := resolveInfluxParam(config.InfluxURL, "INFLUX_URL")
	token := resolveInfluxParam(config.InfluxToken, "INFLUX_TOKEN")
	org := resolveInfluxParam(config.InfluxOrg, "INFLUX_ORG")
	bucket := resolveInfluxParam(config.InfluxBucket, "INFLUX_BUCKET")

	if url == "" || token == "" || org == "" || bucket == "" {
		return EnergyStats{}, ErrInfluxNotConfigured
	}

	client := influxdb2.NewClient(url, token)
	defer client.Close()

	// One-pass online statistics via Flux reduce.
	// The result is a single row with columns "n", "sum", "sum2".
	flux := fmt.Sprintf(`
from(bucket: %q)
  |> range(start: -%s)
  |> filter(fn: (r) => r._measurement == "energy_sample")
  |> filter(fn: (r) => r.function_name == %q)
  |> filter(fn: (r) => r._field == "invocation_joule")
  |> group()
  |> reduce(
       identity: {n: 0, sum: 0.0, sum2: 0.0},
       fn: (r, accumulator) => ({
         n:    accumulator.n    + 1,
         sum:  accumulator.sum  + r._value,
         sum2: accumulator.sum2 + r._value * r._value
       })
  )
`, bucket, window, functionName)

	queryAPI := client.QueryAPI(org)
	result, err := queryAPI.Query(ctx, flux)
	if err != nil {
		return EnergyStats{}, fmt.Errorf("influx query: %w", err)
	}
	defer result.Close()

	var stats EnergyStats
	for result.Next() {
		vals := result.Record().Values()
		n := toInt64(vals["n"])
		if n <= 0 {
			continue
		}
		sum := toFloat64(vals["sum"])
		sum2 := toFloat64(vals["sum2"])
		mean := sum / float64(n)
		// population variance: E[X²] - (E[X])²
		variance := sum2/float64(n) - mean*mean
		if variance < 0 {
			variance = 0
		}
		stats = EnergyStats{
			Mean:   mean,
			Stddev: math.Sqrt(variance),
			Count:  n,
		}
	}
	if err := result.Err(); err != nil {
		return EnergyStats{}, fmt.Errorf("influx query result: %w", err)
	}

	return stats, nil
}

// toFloat64 converts common numeric interface types to float64.
func toFloat64(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case int:
		return float64(x)
	}
	return 0
}

// toInt64 converts common numeric interface types to int64.
func toInt64(v interface{}) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	}
	return 0
}
