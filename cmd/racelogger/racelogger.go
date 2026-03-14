package racelogger

import (
	"github.com/spf13/cobra"
)

func NewRaceloggerCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "racelogger",
		Short: "commands to fetch data for racelogger",
		Long:  ``,
	}
	cmd.AddCommand(NewCarDataCommand())

	return &cmd
}
