package check

import (
	"github.com/spf13/cobra"
)

func NewCheckCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "check",
		Short: "commands related to checking irdata functionality",
		Long:  ``,
	}
	cmd.AddCommand(NewCheckRateLimitCommand())

	return &cmd
}
