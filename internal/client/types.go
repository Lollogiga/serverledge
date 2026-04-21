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
	// CIZoneOverride sovrascrive la zona ElectricityMaps per questa singola invocazione.
	// Se vuoto, viene usata la zona da serverledge-conf.yaml.
	CIZoneOverride string `json:"ci_zone_override,omitempty"`

	// CIOverride, se > 0, usa direttamente questo valore di carbon intensity
	// (gCO2eq/kWh) saltando la query a ElectricityMaps. Usato negli esperimenti
	// per alimentare il server con CI letta da file CSV storici riproducibili.
	CIOverride float64 `json:"ci_override,omitempty"`

	// LambdaOverride, se non nil, impone direttamente il valore di λ ∈ [0,1].
	// Usato per le politiche di baseline negli esperimenti:
	//   0.0 = sempre la variante più efficiente (min energia)
	//   0.5 = sempre la variante di compromesso (metà fronte)
	//   1.0 = sempre la variante più accurata (min errore)
	LambdaOverride *float64 `json:"lambda_override,omitempty"`

	// BetaOverride, se non nil, sovrascrive il parametro β dell'UCB exploration.
	// β=0 disabilita l'esplorazione UCB per questa invocazione.
	BetaOverride *float64 `json:"beta_override,omitempty"`
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
