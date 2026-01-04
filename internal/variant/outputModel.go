package variant

type OutputModel struct {
	Type string `json:"type"` // "error" | "quality"

	ErrorEstimate *float64 `json:"error_estimate,omitempty"`
	Quality       *string  `json:"quality,omitempty"`
}
