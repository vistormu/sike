package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"sike/domain/clock"
	"sike/domain/model"
	"sike/domain/service"
)

func opTaskCmd() *cobra.Command {
	task := &cobra.Command{
		Use:   "task",
		Short: "tasks: done means deleted",
	}

	var area, when, date string
	var at []string
	var notify bool
	add := &cobra.Command{
		Use:   "add <user-id> <name>",
		Short: "add a task, on top of its place in time",
		Example: `  sike op task add 1 "Fix bike" --area Home
  sike op task add 1 "Learn piano" --area lavender --when someday
  sike op task add 1 Gym --area Body --when weekly --at "mon,wed 18:00-19:30"
  sike op task add 1 Dentist --area Admin --when date --date "15/10 11:00-12:00" --notify
  sike op task add 1 Taxes --area Admin --when date --date "due 20/10"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := parseID(args[0])
			if err != nil {
				return err
			}
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			color, err := svc.areaColor(cmd, userID, area)
			if err != nil {
				return err
			}
			t := model.Task{UserID: userID, Name: args[1], Color: color, When: model.When(when)}
			if t.Places, err = parsePlaces(at); err != nil {
				return err
			}
			if date != "" {
				if t.Date, err = parseDates(date, notify); err != nil {
					return err
				}
			}
			t, err = svc.tasks.Add(cmd.Context(), t)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d  %s\n", t.ID, t.Name)
			return nil
		},
	}
	add.Flags().StringVar(&area, "area", "", "the area: its color or its name (required)")
	add.Flags().StringVar(&when, "when", string(model.Pending), "weekly, date, pending or someday")
	add.Flags().StringArrayVar(&at, "at", nil, `a weekly time, "mon,wed 18:00-19:30" (repeat it for more)`)
	add.Flags().StringVar(&date, "date", "", `a date: "15/10 11:00-12:00", "15/10" (all day), "16/10..18/10", "due 20/10 [14:00]"`)
	add.Flags().BoolVar(&notify, "notify", false, "a dated task: notify at its start")
	_ = add.MarkFlagRequired("area")

	list := &cobra.Command{
		Use:   "list <user-id>",
		Short: "list a person's tasks, as the tabs order them",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := parseID(args[0])
			if err != nil {
				return err
			}
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			areas, err := svc.areas.List(cmd.Context(), userID)
			if err != nil {
				return err
			}
			names := map[string]string{}
			for _, a := range areas {
				names[a.Color] = a.Name
			}
			tasks, err := svc.tasks.List(cmd.Context(), userID)
			if err != nil {
				return err
			}
			for _, t := range tasks {
				fmt.Fprintf(cmd.OutOrStdout(), "%d  %-8s  %-8s  %s\n", t.ID, t.When, names[t.Color], t.Name)
				if t.Date != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "     %s\n", formatDates(*t.Date))
				}
				if t.When != model.Weekly {
					continue
				}
				places, err := svc.tasks.Placements(cmd.Context(), userID, t.ID)
				if err != nil {
					return err
				}
				for _, p := range places {
					fmt.Fprintf(cmd.OutOrStdout(), "     %s\n", formatPlace(p))
				}
			}
			return nil
		},
	}

	rm := &cobra.Command{
		Use:   "rm <user-id> <task-id>",
		Short: "delete a task: it is done (restore undoes the last one)",
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
			return svc.tasks.Remove(cmd.Context(), userID, id)
		},
	}

	restore := &cobra.Command{
		Use:   "restore <user-id> <task-id>",
		Short: "undo a delete: only a person's last one can be undone",
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
			return svc.tasks.Restore(cmd.Context(), userID, id)
		},
	}

	move := &cobra.Command{
		Use:     "move <user-id> <task-id> <when>",
		Short:   "put a task in pending or someday, on top (weekly needs times: edit)",
		Example: `  sike op task move 1 4 someday`,
		Args:    cobra.ExactArgs(3),
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
			return svc.tasks.Move(cmd.Context(), userID, id, model.When(args[2]))
		},
	}

	var name, editArea, editWhen, editDate string
	var editAt []string
	var editNotify bool
	edit := &cobra.Command{
		Use:   "edit <user-id> <task-id>",
		Short: "change a task's name, area, when, weekly times or dates; a new when puts it on top there",
		Example: `  sike op task edit 1 4 --name "Fix the bike"
  sike op task edit 1 4 --area Fun --when someday
  sike op task edit 1 4 --when weekly --at "sat 10:00-12:00"
  sike op task edit 1 4 --when date --date "due 20/10"`,
		Args: cobra.ExactArgs(2),
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
			t, err := svc.tasks.Get(cmd.Context(), userID, id)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				t.Name = name
			}
			if cmd.Flags().Changed("area") {
				if t.Color, err = svc.areaColor(cmd, userID, editArea); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("when") {
				t.When = model.When(editWhen)
			}
			if cmd.Flags().Changed("at") {
				if t.Places, err = parsePlaces(editAt); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("date") {
				if t.Date, err = parseDates(editDate, editNotify); err != nil {
					return err
				}
			} else if cmd.Flags().Changed("notify") && t.Date != nil {
				t.Date.Notify = editNotify
			}
			t, err = svc.tasks.Edit(cmd.Context(), t)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d  %s  %s  %s\n", t.ID, t.When, t.Color, t.Name)
			return nil
		},
	}
	edit.Flags().StringVar(&name, "name", "", "the new name")
	edit.Flags().StringVar(&editArea, "area", "", "the new area: its color or its name")
	edit.Flags().StringVar(&editWhen, "when", "", "weekly, date, pending or someday")
	edit.Flags().StringArrayVar(&editAt, "at", nil, `replaces the weekly times: "mon,wed 18:00-19:30" (repeat it for more)`)
	edit.Flags().StringVar(&editDate, "date", "", `replaces the dates: "15/10 11:00-12:00", "15/10", "16/10..18/10", "due 20/10 [14:00]"`)
	edit.Flags().BoolVar(&editNotify, "notify", false, "a dated task: notify at its start")
	edit.MarkFlagsOneRequired("name", "area", "when", "at", "date", "notify")

	task.AddCommand(add, list, edit, rm, restore, move)
	return task
}

type taskSvc struct {
	tasks *service.Tasks
	areas *service.Areas
}

func taskServices(cmd *cobra.Command) (taskSvc, func(), error) {
	st, done, err := openStore(cmd)
	if err != nil {
		return taskSvc{}, nil, err
	}
	return taskSvc{tasks: service.NewTasks(st), areas: service.NewAreas(st)}, done, nil
}

// areaColor finds a person's area by its color or its name, ignoring case.
func (s taskSvc) areaColor(cmd *cobra.Command, userID int64, area string) (string, error) {
	areas, err := s.areas.List(cmd.Context(), userID)
	if err != nil {
		return "", err
	}
	var names []string
	for _, a := range areas {
		if strings.EqualFold(area, a.Color) || strings.EqualFold(area, a.Name) {
			return a.Color, nil
		}
		names = append(names, a.Name)
	}
	return "", fmt.Errorf("no area %q: pick one of %s, or a color", area, strings.Join(names, ", "))
}

// parseDates reads a dated task's --date, dates day first (or YYYY-MM-DD):
// "15/10 11:00-12:00" one day, timed; "15/10" one day, all day;
// "16/10..18/10" (or "16/10 18:00..18/10 09:00") a stretch; "due 20/10" or
// "due 20/10 14:00" a deadline. dates without a year are the nearest.
func parseDates(spec string, notify bool) (*model.Dates, error) {
	loc, err := location()
	if err != nil {
		return nil, err
	}
	today := clock.Today(time.Now(), loc)
	bad := fmt.Errorf("date %q: want \"15/10 11:00-12:00\", \"15/10\", \"16/10..18/10\" or \"due 20/10 14:00\"", spec)
	d := model.Dates{Start: clock.NoTime, End: clock.NoTime, Notify: notify}
	// one "date [time]" or "date [start-end]"
	day := func(s string) (clock.Date, clock.Time, clock.Time, error) {
		f := strings.Fields(s)
		if len(f) == 0 || len(f) > 2 {
			return clock.Date{}, 0, 0, bad
		}
		date, err := clock.ReadDate(f[0], today)
		if err != nil {
			return clock.Date{}, 0, 0, err
		}
		start, end := clock.NoTime, clock.NoTime
		if len(f) == 2 {
			a, b, span := strings.Cut(f[1], "-")
			if start, err = clock.ReadTime(a); err != nil {
				return clock.Date{}, 0, 0, err
			}
			if span {
				if end, err = clock.ReadTime(b); err != nil {
					return clock.Date{}, 0, 0, err
				}
			}
		}
		return date, start, end, nil
	}
	spec = strings.TrimSpace(spec)
	switch {
	case strings.HasPrefix(strings.ToLower(spec), "due "):
		var at clock.Time
		if d.To, at, _, err = day(spec[4:]); err != nil {
			return nil, err
		}
		d.End = at
	case strings.Contains(spec, ".."):
		from, to, _ := strings.Cut(spec, "..")
		if d.From, d.Start, _, err = day(from); err != nil {
			return nil, err
		}
		if d.To, d.End, _, err = day(to); err != nil {
			return nil, err
		}
	default:
		if d.From, d.Start, d.End, err = day(spec); err != nil {
			return nil, err
		}
		d.To = d.From
	}
	return &d, nil
}

// formatDates prints a dated task's dates as parseDates reads them.
func formatDates(d model.Dates) string {
	at := func(t clock.Time) string {
		if t == clock.NoTime {
			return ""
		}
		return " " + t.String()
	}
	var s string
	switch {
	case d.Deadline():
		s = "due " + d.To.String() + at(d.End)
	case d.From == d.To && d.Start != clock.NoTime:
		s = d.From.String() + " " + d.Start.String() + "-" + d.End.String()
	case d.From == d.To:
		s = d.From.String()
	default:
		s = d.From.String() + at(d.Start) + ".." + d.To.String() + at(d.End)
	}
	if d.Notify {
		s += "  notify"
	}
	return s
}

func parsePlaces(specs []string) ([]model.Placement, error) {
	var places []model.Placement
	for _, spec := range specs {
		p, err := parsePlace(spec)
		if err != nil {
			return nil, err
		}
		places = append(places, p)
	}
	return places, nil
}

func parseIDs(a, b string) (int64, int64, error) {
	x, err := parseID(a)
	if err != nil {
		return 0, 0, err
	}
	y, err := parseID(b)
	return x, y, err
}
