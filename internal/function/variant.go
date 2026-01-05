package function

type EnergyProfile struct {
	ColdStartJoule  float64 `json:"cold_start_joule"`
	InvocationJoule float64 `json:"invocation_joule"`
}

type OutputModel struct {
	Type          string   `json:"type"` // "error" | "quality"
	ErrorEstimate *float64 `json:"error_estimate,omitempty"`
	Quality       *string  `json:"quality,omitempty"`
}

type Variant struct {
	ID         string `json:"id"`
	Runtime    string `json:"runtime"`
	EntryPoint string `json:"entry_point"`

	Src string `json:"src"`

	TarCode string `json:"-"`

	Energy EnergyProfile `json:"energy"`
	Output OutputModel   `json:"output"`

	IsApproximate bool `json:"is_approximate"`
}
