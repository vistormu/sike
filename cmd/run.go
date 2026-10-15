package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"sike/adapter/store"
	"sike/view/http/server"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "start the web server",
		Long: `start the web server. SIKE_ADDR sets the listen address (default :8090),
SIKE_DB the database file (default sike.db), created and migrated on boot.`,
		Example: `  sike run
  SIKE_ADDR=:3000 SIKE_DB=/data/sike.db sike run`,
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
			log.Printf("db %s at version %d", path, v)

			srv, err := server.New(server.Config{
				Addr: getenv("SIKE_ADDR", ":8090"),
			})
			if err != nil {
				return err
			}
			return srv.Run(cmd.Context())
		},
	}
}
