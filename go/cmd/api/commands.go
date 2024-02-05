package main

import (
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/pkg/streaminterface"
)

type Commands struct {
	Years interface {
		Create(*commands.Year) error
		Update(*commands.Year) error
		Delete(*commands.Year) error
	}
}

func NewCommands(stream streaminterface.Publisher, models data.Models) Commands {
	return Commands{
		Years: commands.NewYear(stream, models.Years),
	}
}
