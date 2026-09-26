package badger

import (
	extBadger "github.com/dgraph-io/badger/v4"
	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/cmd/config"
	"github.com/mpapenbr/irdata/log"
)

func NewBadgerCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "badger",
		Short: "commands related to badger",
		Long:  ``,
	}

	cmd.AddCommand(NewListKeysCommand())
	cmd.AddCommand(NewDeleteKeysCommand())
	return &cmd
}

func initBadger() (*extBadger.DB, error) {
	var dbErr error
	var db *extBadger.DB
	db, dbErr = extBadger.Open(extBadger.DefaultOptions(config.CacheDir))
	if dbErr != nil {
		log.Error("failed to open cache database", log.ErrorField(dbErr))
		return nil, dbErr
	}
	return db, nil
}
