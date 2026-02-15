package populate

import (
	"github.com/spf13/cobra"
)

func NewPopulateCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "populate",
		Short: "commands related to populating data",
		Long:  ``,
	}

	// cmd.AddCommand(NewLoginCommand())
	return &cmd
}
