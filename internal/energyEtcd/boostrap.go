package energyEtcd

import "context"

// BootstrapIfMissing inizializza lo stato energetico
// usando le stime offline del JSON delle varianti.
func (s *Store) BootstrapIfMissing(
	ctx context.Context,
	functionName string,
	variantID string,
	coldEst float64,
	invocationEst float64,
) error {

	stats, err := s.GetEnergyStats(ctx, functionName, variantID)
	if err != nil {
		return err
	}

	if stats != nil {
		return nil // già inizializzato
	}

	initial := &EnergyStats{
		ColdStartEstimate:  coldEst,
		InvocationEstimate: invocationEst,
		ColdSamples:        0,
		InvocationSamples:  0,
	}

	return s.PutEnergyStats(ctx, functionName, variantID, initial)
}
