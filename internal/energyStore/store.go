package energyStore

import (
	_ "context"
	_ "encoding/json"
	_ "time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Store incapsula l'accesso a etcd
type Store struct {
	client *clientv3.Client
}

// NewStore crea un nuovo energy store
func NewStore(client *clientv3.Client) *Store {
	return &Store{client: client}
}
