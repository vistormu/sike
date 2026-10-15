package store

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"sike/domain/clock"
	"sike/domain/model"
)

// dateRow is task_dates as sql has it: NULL for a missing start or time.
type dateRow struct {
	TaskID    int64          `db:"task_id"`
	StartDate sql.NullString `db:"start_date"`
	StartTime sql.NullInt64  `db:"start_time"`
	EndDate   string         `db:"end_date"`
	EndTime   sql.NullInt64  `db:"end_time"`
	Notify    bool           `db:"notify"`
}

func toRow(taskID int64, d model.Dates) dateRow {
	r := dateRow{TaskID: taskID, EndDate: d.To.String(), Notify: d.Notify}
	if !d.From.IsZero() {
		r.StartDate = sql.NullString{String: d.From.String(), Valid: true}
	}
	if d.Start != clock.NoTime {
		r.StartTime = sql.NullInt64{Int64: int64(d.Start), Valid: true}
	}
	if d.End != clock.NoTime {
		r.EndTime = sql.NullInt64{Int64: int64(d.End), Valid: true}
	}
	return r
}

func (r dateRow) dates() (model.Dates, error) {
	d := model.Dates{Start: clock.NoTime, End: clock.NoTime, Notify: r.Notify}
	var err error
	if d.To, err = clock.ParseDate(r.EndDate); err != nil {
		return d, err
	}
	if r.StartDate.Valid {
		if d.From, err = clock.ParseDate(r.StartDate.String); err != nil {
			return d, err
		}
	}
	if r.StartTime.Valid {
		d.Start = clock.Time(r.StartTime.Int64)
	}
	if r.EndTime.Valid {
		d.End = clock.Time(r.EndTime.Int64)
	}
	return d, nil
}

func insertDates(ctx context.Context, tx *sqlx.Tx, taskID int64, d *model.Dates) error {
	if d == nil {
		return nil
	}
	_, err := tx.NamedExecContext(ctx,
		`INSERT INTO task_dates (task_id, start_date, start_time, end_date, end_time, notify)
		VALUES (:task_id, :start_date, :start_time, :end_date, :end_time, :notify)`, toRow(taskID, *d))
	return err
}

const dateCols = `d.task_id, d.start_date, d.start_time, d.end_date, d.end_time, d.notify`

// datesOf is the dates of a person's dated tasks, by task id.
func (s *Store) datesOf(ctx context.Context, userID int64) (map[int64]*model.Dates, error) {
	var rows []dateRow
	if err := s.db.SelectContext(ctx, &rows,
		`SELECT `+dateCols+` FROM task_dates d JOIN tasks t ON t.id = d.task_id WHERE t.user_id = ?`, userID); err != nil {
		return nil, err
	}
	out := make(map[int64]*model.Dates, len(rows))
	for _, r := range rows {
		d, err := r.dates()
		if err != nil {
			return nil, err
		}
		out[r.TaskID] = &d
	}
	return out, nil
}
