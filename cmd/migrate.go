package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"sike/adapter/store"
)

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "apply pending database migrations",
		Long: `apply pending database migrations to SIKE_DB (default sike.db) and print the
resulting version. run also migrates on boot; this is for doing it by hand.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := getenv("SIKE_DB", "sike.db")
			db, err := store.Open(path)
			if err != nil {
				return err
			}
			defer db.Close()
			v, err := store.Migrate(cmd.Context(), db)
			if err != nil {
				return err
			}
			fmt.Printf("%s at version %d\n", path, v)
			return nil
		},
	}
}
