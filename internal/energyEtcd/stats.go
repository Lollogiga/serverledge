package energyEtcd

import "time"

type EnergyStats struct {
	// Stima del costo di cold start/invocaziome
	ColdStartEstimate  float64 `json:"cold_start_estimate"`
	InvocationEstimate float64 `json:"invocation_estimate"`
	//Numero di campioni usati per stimare il cold start/invocazione
	ColdSamples       int `json:"cold_samples"`
	InvocationSamples int `json:"invocation_samples"`
	//Timestamp ultimo aggiornamento
	LastUpdate time.Time `json:"last_updated"`
}
