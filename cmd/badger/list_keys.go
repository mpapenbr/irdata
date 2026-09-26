package badger

import (
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/spf13/cobra"
)

func NewListKeysCommand() *cobra.Command {
	runner := &listKeysRunner{}

	cmd := cobra.Command{
		Use:   "keys",
		Short: "list keys in the badger database",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.run()
		},
	}
	cmd.Flags().StringVarP(&runner.filter, "filter", "f", "", "filter keys by prefix")
	cmd.Flags().BoolVarP(&runner.bundleEndpoints, "bundle-endpoints", "b", false,
		"bundle endpoints together")

	return &cmd
}

type (
	listKeysRunner struct {
		filter          string
		bundleEndpoints bool
	}
)

func (r *listKeysRunner) run() error {
	db, err := initBadger()
	if err != nil {
		return err
	}
	defer db.Close()
	out := make([]string, 0)
	seen := make(map[string]struct{})

	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := string(item.Key())
			//nolint:nestif // ok by design
			if r.filter == "" || strings.HasPrefix(k, r.filter) {
				if r.bundleEndpoints {
					idx := strings.Index(k, "?")
					if idx != -1 {
						k = k[:idx]
					}
					if _, exists := seen[k]; exists {
						continue
					}
					seen[k] = struct{}{}
				}
				out = append(out, k)
			}
		}

		for _, k := range out {
			fmt.Printf("%s\n", k)
		}
		return nil
	})
	return err
}
