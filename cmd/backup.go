package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"sike/adapter/store"
)

func backupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup <file>",
		Short: "write a consistent copy of SIKE_DB to file, while the server runs",
		Long: `write a consistent copy of SIKE_DB (default sike.db) to file, which must not
exist, while the server runs: the homelab takes one before a version change, and
the nightly backup can. restore: stop the server, put the copy in SIKE_DB's place
(drop its -wal and -shm files), start it.`,
		Example: `  sike backup /data/backup/v0.1.0.db`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, _, _, err := openDB(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			if err := store.Backup(cmd.Context(), db, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "backed up to %s\n", args[0])
			return nil
		},
	}
}
