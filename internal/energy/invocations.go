package energy

import (
	"sync"
	"sync/atomic"
)

// Contatore cumulativo per container.
// Questo è l'unico elemento "energy-related" nel critical path (ma è O(1) e atomico).
var (
	invMu     sync.RWMutex
	invCounts = map[string]*uint64{}
)

func getInvPtr(containerID string) *uint64 {
	invMu.RLock()
	p, ok := invCounts[containerID]
	invMu.RUnlock()
	if ok {
		return p
	}

	invMu.Lock()
	defer invMu.Unlock()

	// double check
	if p, ok = invCounts[containerID]; ok {
		return p
	}

	var v uint64
	invCounts[containerID] = &v
	return &v
}

// IncInvocation incrementa il numero totale di invocazioni eseguite dal container.
func IncInvocation(containerID string) {
	p := getInvPtr(containerID)
	atomic.AddUint64(p, 1)
}

// LoadInvocations legge il contatore cumulativo N(container).
func LoadInvocations(containerID string) uint64 {
	invMu.RLock()
	p, ok := invCounts[containerID]
	invMu.RUnlock()
	if !ok {
		return 0
	}
	return atomic.LoadUint64(p)
}

// ResetInvocations rimuove il contatore quando un container viene eliminato.
func ResetInvocations(containerID string) {
	invMu.Lock()
	defer invMu.Unlock()
	delete(invCounts, containerID)
}
