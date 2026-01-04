package function

// Variant represents a concrete executable implementation of a logical function.
// Variants may differ in runtime, performance, energy consumption and accuracy.
type Variant struct {
	ID string `json:"id"`

	Runtime    string `json:"runtime"`     // e.g., python310, c-native
	EntryPoint string `json:"entry_point"` // handler or binary entry point

	EnergyEstimateJ float64 `json:"energy_estimate_j"` // estimated energy cost (Joules)
	ErrorEstimate   float64 `json:"error_estimate"`    // approximation error (0 = exact)

	IsApproximate bool `json:"is_approximate"` // whether this variant is approximate
}
