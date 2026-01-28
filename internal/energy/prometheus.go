package energy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var (
	// ErrNoData: Prometheus/Kepler non ha ancora una serie per quel container (caso normale all'inizio)
	ErrNoData = errors.New("no kepler data")

	// ErrTransient: errore "transiente" (es. endpoint instabile, decode fallito, status non-200, response incoerente).
	// Per tua specifica: se succede, il worker deve saltare l'intero ciclo.
	ErrTransient = errors.New("transient prometheus/kepler error")
)

type PrometheusReader interface {
	ReadContainerJoule(containerID string) (float64, error)
}

type PrometheusClient struct {
	BaseURL string
	Timeout time.Duration
}

func NewPrometheusClient(baseURL string) *PrometheusClient {
	return &PrometheusClient{
		BaseURL: baseURL,
		Timeout: 2 * time.Second,
	}
}

type promResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func (p *PrometheusClient) ReadContainerJoule(containerID string) (float64, error) {
	// Energia cumulativa CPU joules del container (Kepler). Rimane cumulativa nel tempo.
	query := fmt.Sprintf(
		`sum(kepler_container_cpu_joules_total{container_id="%s",zone="core"})`,
		containerID,
	)

	u := fmt.Sprintf("%s/api/v1/query?query=%s", p.BaseURL, url.QueryEscape(query))

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, ErrTransient
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, ErrTransient
	}

	var pr promResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return 0, ErrTransient
	}

	if pr.Status != "success" {
		return 0, ErrTransient
	}

	if len(pr.Data.Result) == 0 {
		return 0, ErrNoData
	}

	// value: [ <timestamp>, "<string_value>" ]
	if len(pr.Data.Result[0].Value) < 2 {
		return 0, ErrTransient
	}

	valStr, ok := pr.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, ErrTransient
	}

	joule, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, ErrTransient
	}

	// Se arrivano NaN/Inf (può succedere con serie “instabili”), trattalo come transiente.
	if math.IsNaN(joule) || math.IsInf(joule, 0) {
		return 0, ErrTransient
	}

	return joule, nil
}
