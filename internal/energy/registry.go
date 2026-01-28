package energy

import (
	"sync"
	"time"
)

type ContainerState struct {
	ContainerID string

	LogicalName string
	VariantName string
	Runtime     string

	// ultimo valore letto da Prometheus (energia cumulativa)
	LastJoule float64
	HasValue  bool

	// contatore invocazioni cumulativo osservato all'ultimo tick
	LastInvocations uint64
	HasInvValue     bool

	// timestamp ultima lettura
	LastRead time.Time
}

var (
	mu         sync.Mutex
	containers = make(map[string]*ContainerState)
)

func RegisterContainer(state *ContainerState) {
	mu.Lock()
	defer mu.Unlock()
	containers[state.ContainerID] = state
}

func UnregisterContainer(containerID string) {
	mu.Lock()
	defer mu.Unlock()

	delete(containers, containerID)
	ResetInvocations(containerID) // cleanup contatore invocazioni
}

func SnapshotContainers() []*ContainerState {
	mu.Lock()
	defer mu.Unlock()

	out := make([]*ContainerState, 0, len(containers))
	for _, c := range containers {
		out = append(out, c)
	}
	return out
}
