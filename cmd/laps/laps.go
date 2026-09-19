package laps

import (
	"github.com/spf13/cobra"
)

func NewLapsCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "laps",
		Short: "commands related to laps",
		Long:  ``,
	}

	cmd.AddCommand(NewLapEventsCommand())

	return &cmd
}
