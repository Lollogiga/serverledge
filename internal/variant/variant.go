package variant

// Variant represents a concrete executable implementation of a logical function.
// Variants may differ in runtime, performance, energy consumption and accuracy.
type Variant struct {
	ID         string `json:"id"`
	Runtime    string `json:"runtime"`
	EntryPoint string `json:"entry_point"`

	Energy EnergyProfile `json:"energy"`
	Output OutputModel   `json:"output"`

	IsApproximate bool `json:"is_approximate"`
}
