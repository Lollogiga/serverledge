package variant

type EnergyProfile struct {
	ColdStartJoule  float64 `json:"cold_start_joule"`
	InvocationJoule float64 `json:"invocation_joule"`
}
