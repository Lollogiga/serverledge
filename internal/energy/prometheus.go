package energy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type PrometheusReader interface {
	ReadContainerJoule(containerID string) (float64, error)
}

type PrometheusClient struct {
	BaseURL string
}

func NewPrometheusClient(baseURL string) *PrometheusClient {
	return &PrometheusClient{BaseURL: baseURL}
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

	query := fmt.Sprintf(
		`sum(kepler_container_cpu_joules_total{container_id="%s",zone="core"})`,
		containerID,
	)

	u := fmt.Sprintf(
		"%s/api/v1/query?query=%s",
		p.BaseURL,
		url.QueryEscape(query),
	)

	req, _ := http.NewRequestWithContext(context.Background(), "GET", u, nil)
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

	val, ok := pr.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, fmt.Errorf("unexpected value type")
	}

	joule, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, err
	}

	return joule, nil
}
