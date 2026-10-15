package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"sike/domain/service"
)

func opAreaCmd() *cobra.Command {
	area := &cobra.Command{
		Use:   "area",
		Short: "areas: a person's seven named colors",
	}

	list := &cobra.Command{
		Use:     "list <user-id>",
		Short:   "list a person's areas, in palette order",
		Example: `  sike op area list 1`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := parseID(args[0])
			if err != nil {
				return err
			}
			areas, done, err := areasService(cmd)
			if err != nil {
				return err
			}
			defer done()
			all, err := areas.List(cmd.Context(), userID)
			if err != nil {
				return err
			}
			for _, a := range all {
				fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %s\n", a.Color, a.Name)
			}
			return nil
		},
	}

	rename := &cobra.Command{
		Use:     "rename <user-id> <color> <name>",
		Short:   "rename one of a person's areas",
		Example: `  sike op area rename 1 sky Job`,
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := parseID(args[0])
			if err != nil {
				return err
			}
			areas, done, err := areasService(cmd)
			if err != nil {
				return err
			}
			defer done()
			return areas.Rename(cmd.Context(), userID, args[1], args[2])
		},
	}

	area.AddCommand(list, rename)
	return area
}

func areasService(cmd *cobra.Command) (*service.Areas, func(), error) {
	st, done, err := openStore(cmd)
	if err != nil {
		return nil, nil, err
	}
	return service.NewAreas(st), done, nil
}

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("id %q is not a number", s)
	}
	return id, nil
}
