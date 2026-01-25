package energyEtcd

import (
	"context"
	"encoding/json"
)

func (s *Store) GetEnergyStats(
	ctx context.Context,
	functionName string,
	variantID string) (*EnergyStats, error) {

	key := EnergyKey(functionName, variantID)
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var stats EnergyStats
	if err := json.Unmarshal(resp.Kvs[0].Value, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}
