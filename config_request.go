package goalx

// ResolveRequest captures per-run overrides that are applied on top of loaded config layers.
type ResolveRequest struct {
	ManualDraft      *Config
	Mode             Mode
	Objective        string
	Preset           string
	Parallel         int
	MasterOverride   *MasterConfig
	ResearchOverride *SessionConfig
	DevelopOverride  *SessionConfig
}
