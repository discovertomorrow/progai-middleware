package llamacpp

// Llama.cpp

type Request struct {
	Prompt           string       `json:"prompt"`
	Temperature      *float32     `json:"temperature,omitempty"`
	TopK             *int         `json:"top_k,omitempty"`
	TopP             *float32     `json:"top_p,omitempty"`
	MinP             *float32     `json:"min_p,omitempty"`
	NPredict         int          `json:"n_predict,omitempty"`
	NKeep            *int         `json:"n_keep,omitempty"`
	Stream           bool         `json:"stream"`
	Stop             []string     `json:"stop,omitempty"`
	TfsZ             *float32     `json:"tfs_z,omitempty"`
	TypicalP         *float32     `json:"typical_p,omitempty"`
	RepeatPenalty    *float32     `json:"repeat_penalty,omitempty"`
	RepeatLastN      *int         `json:"repleat_last_n,omitempty"`
	PenalizeNl       *bool        `json:"penalize_nl,omitempty"`
	PresencePenalty  *float32     `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32     `json:"frequency_penalty,omitempty"`
	PenaltyPrompt    *string      `json:"penalty_prompt,omitempty"`
	Mirostat         *int         `json:"mirostat,omitempty"`
	MirostatTau      *float32     `json:"mirostat_tau,omitempty"`
	MirostatEta      *float32     `json:"mirostat_eta,omitempty"`
	Grammar          *string      `json:"grammar,omitempty"` // TODO: check type
	Seed             *int         `json:"seed,omitempty"`
	Slot             int          `json:"id_slot"`
	CachePrompt      bool         `json:"cache_prompt"`
	LogitBias        [][2]float64 `json:"logit_bias"`
	// TODO: logit_bias
	// TODO: n_probs
	// TODO: image_data
	// ignore_eos omitted
	// system_prompt omitted
}
