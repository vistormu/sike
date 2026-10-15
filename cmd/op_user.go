package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"sike/domain/service"
)

func opUserCmd() *cobra.Command {
	user := &cobra.Command{
		Use:   "user",
		Short: "people: each one is a separate world",
	}

	add := &cobra.Command{
		Use:     "add <name> <emoji>",
		Short:   "add a person",
		Example: `  sike op user add You 🦊`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			users, done, err := usersService(cmd)
			if err != nil {
				return err
			}
			defer done()
			u, err := users.Add(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d  %s  %s\n", u.ID, u.Emoji, u.Name)
			return nil
		},
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "list people, oldest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			users, done, err := usersService(cmd)
			if err != nil {
				return err
			}
			defer done()
			all, err := users.List(cmd.Context())
			if err != nil {
				return err
			}
			for _, u := range all {
				fmt.Fprintf(cmd.OutOrStdout(), "%d  %s  %s\n", u.ID, u.Emoji, u.Name)
			}
			return nil
		},
	}

	var yes bool
	rm := &cobra.Command{
		Use:   "rm <id>",
		Short: "delete a person and everything theirs",
		Args:  cobra.ExactArgs(1),
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
			if !yes && !confirm(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("delete %s %s and everything theirs?", u.Emoji, u.Name)) {
				return fmt.Errorf("nothing deleted")
			}
			return users.Remove(cmd.Context(), id)
		},
	}
	rm.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask")

	user.AddCommand(add, list, rm)
	return user
}

func usersService(cmd *cobra.Command) (*service.Users, func(), error) {
	st, done, err := openStore(cmd)
	if err != nil {
		return nil, nil, err
	}
	return service.NewUsers(st), done, nil
}
