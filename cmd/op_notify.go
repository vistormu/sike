package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"sike/adapter/ntfy"
	"sike/domain/port"
	"sike/domain/service"
)

func opNotifyCmd() *cobra.Command {
	notify := &cobra.Command{
		Use:   "notify",
		Short: "notifications: ntfy (SIKE_NTFY_URL, SIKE_NTFY_TOKEN)",
	}

	test := &cobra.Command{
		Use:     "test <user-id>",
		Short:   "send a test message to a person's ntfy topic",
		Example: `  sike op notify test 1`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			users, done, err := usersService(cmd)
			if err != nil {
				return err
			}
			defer done()
			u, err := users.Get(cmd.Context(), id)
			if err != nil {
				return err
			}
			if err := service.NewNotifier(notifierPort()).Test(cmd.Context(), u, u.Topic); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "sent to %s\n", u.Topic)
			return nil
		},
	}

	notify.AddCommand(test)
	return notify
}

// notifierPort is ntfy at SIKE_NTFY_URL with SIKE_NTFY_TOKEN; none when the
// url is unset: notifications off.
func notifierPort() port.Notifier {
	url := getenv("SIKE_NTFY_URL", "")
	if url == "" {
		return nil
	}
	return ntfy.New(url, getenv("SIKE_NTFY_TOKEN", ""))
}
