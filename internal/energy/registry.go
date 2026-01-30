package energy

import (
	"log"
	"sync"
	"time"
)

var (
	mu         sync.Mutex
	containers = make(map[string]*ContainerState)
)

type ContainerState struct {
	ContainerID string

	// Identità
	FunctionName string
	LogicalName  string
	VariantID    string

	// Energia / contatori
	LastJoule          float64
	LastInvocations    uint64
	PendingEnergyJ     float64
	PendingInvocations uint64
	HasValue           bool
	LastRead           time.Time
}

// RegisterContainer registers or updates a container state.
// It MUST be called with the real Function that generated the container.
func RegisterContainer(state *ContainerState) {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := containers[state.ContainerID]; ok {
		// ⚠️ NON aggiornare FunctionName / VariantID
		// l’identità del container è IMMUTABILE
		log.Printf(
			"[energy][registry] container=%s already registered (identity preserved)",
			state.ContainerID,
		)
		return
	}

	log.Printf(
		"[energy][registry] new container=%s fn=%s logical=%s variant=%s",
		state.ContainerID,
		state.FunctionName,
		state.LogicalName,
		state.VariantID,
	)

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
		log.Printf(
			"[DEBUG][snapshot] id=%s logical=%s variant=%s function=%s",
			c.ContainerID,
			c.LogicalName,
			c.VariantID,
			c.FunctionName,
		)
		out = append(out, c)
	}
	return out
}
