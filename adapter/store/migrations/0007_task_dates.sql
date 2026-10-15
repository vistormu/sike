-- +goose Up
-- when a dated task happens, one row per task with "when" = 'date'. no start
-- date: a deadline on the end date. dates are YYYY-MM-DD, times minutes after
-- midnight (NULL: none). it goes with its task, through delete and undo.
CREATE TABLE task_dates (
    task_id    INTEGER PRIMARY KEY REFERENCES tasks (id) ON DELETE CASCADE,
    start_date TEXT,
    start_time INTEGER CHECK (start_time BETWEEN 0 AND 1439),
    end_date   TEXT    NOT NULL,
    end_time   INTEGER CHECK (end_time BETWEEN 0 AND 1440),
    notify     INTEGER NOT NULL DEFAULT 0,
    CHECK (start_date IS NULL OR start_date <= end_date),
    CHECK (start_date IS NOT NULL OR start_time IS NULL)
);

-- +goose Down
DROP TABLE task_dates;
