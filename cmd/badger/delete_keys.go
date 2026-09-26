package badger

import (
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/spf13/cobra"
)

func NewDeleteKeysCommand() *cobra.Command {
	runner := &deleteKeysRunner{}

	cmd := cobra.Command{
		Use:   "delete",
		Short: "delete keys in the badger database",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.run()
		},
	}
	cmd.Flags().StringVarP(&runner.filter, "filter", "f", "", "filter keys by prefix")

	return &cmd
}

type (
	deleteKeysRunner struct {
		filter string
	}
)

func (r *deleteKeysRunner) run() error {
	db, err := initBadger()
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := string(item.Key())

			if r.filter == "" || strings.HasPrefix(k, r.filter) {
				if delErr := txn.Delete([]byte(k)); delErr != nil {
					return delErr
				}
			}
		}

		return nil
	})
	return err
}
