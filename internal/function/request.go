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

	AllowApprox    bool
	MaxEnergyJoule *float64
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

type VariantSchedulingReport struct {
	LogicalName      string `json:"logical_name,omitempty"`
	InvokedFunction  string `json:"invoked_function,omitempty"`
	SelectedFunction string `json:"selected_function,omitempty"`
	VariantID        string `json:"variant_id,omitempty"`

	AllowApprox    bool    `json:"allow_approx"`
	MaxEnergyJoule float64 `json:"max_energy_joule,omitempty"`

	EstimatedEnergy float64 `json:"estimated_energy_joule,omitempty"`
	WarmHint        bool    `json:"warm_hint"`

	AccuracyScore  float64 `json:"accuracy_score"`
	DecisionReason string  `json:"decision_reason,omitempty"`
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
