package client

import (
	"github.com/serverledge-faas/serverledge/internal/function"
)

// InvocationRequest is an external invocation of a function (from API or CLI)
type InvocationRequest struct {
	Params          map[string]interface{}
	QoSClass        int64
	QoSMaxRespT     float64
	CanDoOffloading bool
	Async           bool
	ReturnOutput    bool

	// QualityWeight drives Pareto-scalarised variant selection:
	// 0.0 → minimise energy, 1.0 → minimise error, nil → no variant selection.
	QualityWeight *float64 `json:"qualityWeight,omitempty"`
}

type PrewarmingRequest struct {
	Function       string
	Instances      int64
	ForceImagePull bool
}

// WorkflowInvocationRequest is an external invocation of a workflow (from API or CLI)
type WorkflowInvocationRequest struct {
	Params          map[string]interface{}
	QoS             function.RequestQoS
	CanDoOffloading bool
	Async           bool
}

type WorkflowCreationRequest struct {
	Name   string // Name of the new workflow
	ASLSrc string // Specification source in Amazon State Language (encoded in Base64)
}
