package irdata

//nolint:tagliatelle // external definition
type (
	LookupResponse struct {
		Lookups []Lookup `json:"lookups,omitempty"`
		Tag     string   `json:"tag,omitempty"`
	}
	Lookup struct {
		LookupType   string        `json:"lookup_type,omitempty"`
		LookupValues []LookupValue `json:"lookup_values,omitempty"`
	}

	LookupValue struct {
		Description string `json:"description,omitempty"`
		Seq         int    `json:"seq,omitempty"`
		Value       string `json:"value,omitempty"`
	}
)
