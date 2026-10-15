package cmd

import (
	"log"
	"time"

	"github.com/spf13/cobra"

	"sike/adapter/store"
	"sike/domain/service"
	"sike/view/http/server"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "start the web server",
		Long: `start the web server. SIKE_ADDR sets the listen address (default :8090),
SIKE_DB the database file (default sike.db), created and migrated on boot,
SIKE_TZ the timezone times are in (default Europe/Madrid).`,
		Example: `  sike run
  SIKE_ADDR=:3000 SIKE_DB=/data/sike.db sike run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, path, v, err := openDB(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			log.Printf("db %s at version %d", path, v)

			loc, err := location()
			if err != nil {
				return err
			}
			st := store.New(db)
			srv, err := server.New(server.Config{
				Addr:     getenv("SIKE_ADDR", ":8090"),
				Users:    service.NewUsers(st),
				Areas:    service.NewAreas(st),
				Tasks:    service.NewTasks(st),
				Moods:    service.NewMoods(st),
				Notifier: service.NewNotifier(notifierPort()),
				Loc:      loc,
				Now:      time.Now,
			})
			if err != nil {
				return err
			}
			return srv.Run(cmd.Context())
		},
	}
}
