package splits

import (
	"github.com/spf13/cobra"
)

func NewSplitsCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "splits",
		Short: "commands related to splits",
		Long:  ``,
	}

	cmd.AddCommand(NewSplitsDataCommand())

	return &cmd
}
