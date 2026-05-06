package klan

import (
	"nathejk.dk/nathejk/table/signup"
)

type repository struct {
	Signup signup.Queries
}

type external func(*repository)

func WithSignup(q signup.Queries) external {
	return func(r *repository) {
		r.Signup = q
	}
}

func NewRepository(es ...external) repository {
	r := repository{}
	for _, with := range es {
		with(&r)
	}
	return r
}
