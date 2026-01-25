package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type PrometheusCollector struct {
	BaseURL string
}

func NewPrometheusCollector(baseURL string) *PrometheusCollector {
	return &PrometheusCollector{BaseURL: baseURL}
}

type promResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// MeasureInvocationJoule misura l'energia CPU (Joule) su finestra fissa 5s
func (p *PrometheusCollector) MeasureInvocationJoule(
	ctx context.Context,
	containerID string,
) (float64, error) {

	const windowSeconds = 5

	query := fmt.Sprintf(
		`increase(kepler_container_cpu_joules_total{container_id="%s"}[%ds])`,
		containerID,
		windowSeconds,
	)

	q := url.Values{}
	q.Set("query", query)

	reqURL := fmt.Sprintf("%s/api/v1/query?%s", p.BaseURL, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var pr promResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return 0, err
	}

	if pr.Status != "success" || len(pr.Data.Result) == 0 {
		return 0, fmt.Errorf("no kepler data for container %s", containerID)
	}

	valStr := pr.Data.Result[0].Value[1].(string)
	joule, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, err
	}

	return joule, nil
}
