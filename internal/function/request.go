package function

import (
	"context"
	"fmt"
	"time"
)

// Request represents a single function invocation, with a ReqId, reference to the Function, parameters and metrics data
type Request struct {
	Ctx     context.Context
	Fun     *Function
	Params  map[string]interface{}
	Arrival time.Time
	RequestQoS
	CanDoOffloading bool
	Async           bool
	ReturnOutput    bool

	// CIZoneOverride, se non vuoto, sostituisce la zona letta da config
	// solo per questa invocazione (utile per test multi-zona).
	CIZoneOverride string
}

type RequestQoS struct {
	Class    int64
	MaxRespT float64
}

type ExecutionReport struct {
	Result         string
	ResponseTime   float64 // time waited by the user to get the output: completion time - arrival time
	IsWarmStart    bool
	InitTime       float64 // time spent sleeping before initializing container
	QueueingTime   float64 // time spent waiting in the queue
	OffloadLatency float64 // time spent offloading the request
	Duration       float64 // execution (service) time
	Output         string

	VariantSchedulingReport *VariantSchedulingReport `json:"variant_scheduling,omitempty"`
}

// ParetoPoint holds the key metrics for one variant on the Pareto front,
// enabling external tools to reconstruct Pareto-front charts.
type ParetoPoint struct {
	FunctionName  string  `json:"function_name"`
	VariantID     string  `json:"variant_id,omitempty"`
	Energy        float64 `json:"energy_joule"`
	ErrorEstimate float64 `json:"error_estimate"`
	NormEnergy    float64 `json:"norm_energy"`
	NormError     float64 `json:"norm_error"`
	Score         float64 `json:"score"`

	// UCB fields (populated when UCB exploration is active)
	UCBExploration bool  `json:"ucb_exploration,omitempty"`  // true if LCB adjusted the energy
	UCBSampleCount int64 `json:"ucb_sample_count,omitempty"` // InfluxDB samples used
}

type VariantSchedulingReport struct {
	LogicalName      string `json:"logical_name,omitempty"`
	InvokedFunction  string `json:"invoked_function,omitempty"`
	SelectedFunction string `json:"selected_function,omitempty"`
	VariantID        string `json:"variant_id,omitempty"`

	// QualityWeight (λ) used during this invocation, derived from CarbonIntensityGCO2.
	QualityWeight float64 `json:"quality_weight"`

	// CarbonIntensityGCO2 is the grid carbon intensity (gCO2eq/kWh) read from
	// ElectricityMaps at the moment of this invocation (0 if unavailable).
	CarbonIntensityGCO2 float64 `json:"carbon_intensity_gco2,omitempty"`

	EstimatedEnergy float64 `json:"estimated_energy_joule,omitempty"`
	WarmHint        bool    `json:"warm_hint"`

	ErrorEstimate  float64 `json:"error_estimate"`
	DecisionReason string  `json:"decision_reason,omitempty"`

	// ParetoFront contains all Pareto-optimal variants considered during
	// selection; useful for plotting energy/error trade-off charts.
	ParetoFront []ParetoPoint `json:"pareto_front,omitempty"`

	// UCBActive is true when at least one variant's energy estimate was
	// adjusted by the UCB/LCB formula during this scheduling decision.
	UCBActive bool `json:"ucb_active,omitempty"`
}

type Response struct {
	Success bool
	ExecutionReport
}

type AsyncResponse struct {
	ReqId string
}

func (r *Request) Id() string {
	return r.Ctx.Value("ReqId").(string)
}

func (r *Request) String() string {
	return fmt.Sprintf("[%s] Rq-%s", r.Fun.Name, r.Id())
}
