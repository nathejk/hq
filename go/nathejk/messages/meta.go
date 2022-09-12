package messages

import "nathejk.dk/nathejk/types"

type MetaID struct {
	ID string `json:"id"`
}

type ByUserMeta struct {
	UserID types.UserID `json:"userId"`
}

type Metadata struct {
	Producer string `json:"producer"`
	Phase    string `json:"phase,omitempty"`
}
