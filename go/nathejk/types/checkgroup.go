package types

type CheckpointScheme string

const (
	CheckpointSchemeFixed    CheckpointScheme = "fixed"
	CheckpointSchemeRelative CheckpointScheme = "relative"
	CheckpointSchemeNone     CheckpointScheme = "none"
)
