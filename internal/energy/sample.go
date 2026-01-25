package energy

import "time"

// EnergySample rappresenta UN campione grezzo (una invocazione)
type EnergySample struct {
	Timestamp time.Time

	LogicalName  string
	VariantName  string
	Runtime      string
	IsWarm       bool
	ExperimentID string

	InvocationJoule float64
	ColdStartJoule  *float64

	DurationMs  int64
	InitTimeMs  int64
	QueueTimeMs int64

	ContainerID string
}
