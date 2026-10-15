package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"

	"sike/adapter/store"
)

// version is set by the image build (-ldflags "-X sike/cmd.version=v0.1.0").
var version = "dev"

func Execute() {
	root := &cobra.Command{
		Use:     "sike",
		Short:   "the week that repeats — a planner for two",
		Version: version,
		// errors are printed once, below, without the usage dump
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(runCmd(), migrateCmd(), backupCmd(), healthCmd(), opCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// getenv is the only place env is read: cmd/ passes values down, nothing below
// touches os.Getenv.
func getenv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

// openDB opens SIKE_DB (default sike.db), creating and migrating it.
func openDB(cmd *cobra.Command) (db *sqlx.DB, path string, version int64, err error) {
	path = getenv("SIKE_DB", "sike.db")
	db, err = store.Open(path)
	if err != nil {
		return nil, "", 0, err
	}
	version, err = store.Migrate(cmd.Context(), db)
	if err != nil {
		db.Close()
		return nil, "", 0, err
	}
	return db, path, version, nil
}

// openStore is openDB for the op commands: the store and a func to close it.
func openStore(cmd *cobra.Command) (*store.Store, func(), error) {
	db, _, _, err := openDB(cmd)
	if err != nil {
		return nil, nil, err
	}
	return store.New(db), func() { db.Close() }, nil
}
