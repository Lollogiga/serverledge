package energyEtcd

import (
	"context"
	"encoding/json"
	"time"
)

func (s *Store) PutEnergyStats(
	ctx context.Context,
	functionName string,
	variantID string,
	stats *EnergyStats,
) error {

	stats.LastUpdate = time.Now().UTC()

	key := EnergyKey(functionName, variantID)
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	_, err = s.client.Put(ctx, key, string(data))
	return err
}
