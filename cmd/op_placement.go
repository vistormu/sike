package cmd

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"sike/domain/clock"
	"sike/domain/model"
)

func opPlacementCmd() *cobra.Command {
	placement := &cobra.Command{
		Use:   "placement",
		Short: "the times a weekly task takes in the week",
	}

	list := &cobra.Command{
		Use:   "list <user-id> <task-id>",
		Short: "a weekly task's times",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, taskID, err := parseIDs(args[0], args[1])
			if err != nil {
				return err
			}
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			places, err := svc.tasks.Placements(cmd.Context(), userID, taskID)
			if err != nil {
				return err
			}
			for _, p := range places {
				fmt.Fprintf(cmd.OutOrStdout(), "%d  %s\n", p.ID, formatPlace(p))
			}
			return nil
		},
	}

	var notifyStart, notifyEnd bool
	add := &cobra.Command{
		Use:   "add <user-id> <task-id> <days> <start-end>",
		Short: "add a time to a weekly task",
		Example: `  sike op placement add 1 4 mon,wed 18:00-19:30 --notify-start
  sike op placement add 1 5 mon-fri 09:00-17:00`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, taskID, err := parseIDs(args[0], args[1])
			if err != nil {
				return err
			}
			p, err := parsePlace(args[2] + " " + args[3])
			if err != nil {
				return err
			}
			p.TaskID, p.NotifyStart, p.NotifyEnd = taskID, notifyStart, notifyEnd
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			p, err = svc.tasks.Place(cmd.Context(), userID, p)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d  %s\n", p.ID, formatPlace(p))
			return nil
		},
	}
	add.Flags().BoolVar(&notifyStart, "notify-start", false, "notify when it starts")
	add.Flags().BoolVar(&notifyEnd, "notify-end", false, "notify when it ends")

	rm := &cobra.Command{
		Use:   "rm <user-id> <placement-id>",
		Short: "remove a time from a weekly task (not its last one)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, id, err := parseIDs(args[0], args[1])
			if err != nil {
				return err
			}
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			return svc.tasks.Unplace(cmd.Context(), userID, id)
		},
	}

	placement.AddCommand(list, add, rm)
	return placement
}

var dayNames = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"} // by time.Weekday

// parsePlace reads "<days> <start>-<end>": days as mon,wed or mon-fri (or
// both, mon-wed,sat), times as HH:MM.
func parsePlace(spec string) (model.Placement, error) {
	days, span, ok := strings.Cut(strings.TrimSpace(spec), " ")
	start, end, ok2 := strings.Cut(strings.TrimSpace(span), "-")
	if !ok || !ok2 {
		return model.Placement{}, fmt.Errorf("time %q: want \"mon,wed 18:00-19:30\"", spec)
	}
	var p model.Placement
	for _, part := range strings.Split(strings.ToLower(days), ",") {
		from, to, isRange := strings.Cut(part, "-")
		a, b := slices.Index(dayNames, from), slices.Index(dayNames, to)
		if !isRange {
			b = a
		}
		if a < 0 || b < 0 {
			return model.Placement{}, fmt.Errorf("days %q: use mon, tue, wed, thu, fri, sat, sun", part)
		}
		// mon-fri walks the week in order; sat-sun wraps past sunday's 0
		for d := a; ; d = (d + 1) % 7 {
			p.Days |= model.DaysOf(time.Weekday(d))
			if d == b {
				break
			}
		}
	}
	var err error
	if p.Start, err = clock.ParseTime(start); err != nil {
		return model.Placement{}, err
	}
	if p.End, err = clock.ParseTime(end); err != nil {
		return model.Placement{}, err
	}
	return p, nil
}

// formatPlace prints a placement as parsePlace reads it, plus its notices.
func formatPlace(p model.Placement) string {
	var days []string
	for _, d := range p.Days.List() {
		days = append(days, dayNames[d])
	}
	s := fmt.Sprintf("%s %s-%s", strings.Join(days, ","), p.Start, p.End)
	if p.NotifyStart {
		s += "  notify start"
	}
	if p.NotifyEnd {
		s += "  notify end"
	}
	return s
}
