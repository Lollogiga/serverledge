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

	// QualityWeight drives Pareto-scalarised variant selection:
	// 0.0 → minimise energy, 1.0 → minimise error, nil → no variant selection.
	QualityWeight *float64
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
}

type VariantSchedulingReport struct {
	LogicalName      string `json:"logical_name,omitempty"`
	InvokedFunction  string `json:"invoked_function,omitempty"`
	SelectedFunction string `json:"selected_function,omitempty"`
	VariantID        string `json:"variant_id,omitempty"`

	// QualityWeight used during this invocation (0 = min energy, 1 = min error).
	QualityWeight float64 `json:"quality_weight"`

	EstimatedEnergy float64 `json:"estimated_energy_joule,omitempty"`
	WarmHint        bool    `json:"warm_hint"`

	ErrorEstimate  float64 `json:"error_estimate"`
	DecisionReason string  `json:"decision_reason,omitempty"`

	// ParetoFront contains all Pareto-optimal variants considered during
	// selection; useful for plotting energy/error trade-off charts.
	ParetoFront []ParetoPoint `json:"pareto_front,omitempty"`
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
