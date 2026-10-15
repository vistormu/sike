-- +goose Up
-- the times a weekly task takes in the week. days is a set, one bit per weekday
-- (sunday = bit 0, as go's time.Weekday); start and end are minutes after
-- midnight, on one day (decisions/0008). a task's placements go with it,
-- through its delete and its undo.
CREATE TABLE placements (
    id           INTEGER PRIMARY KEY,
    task_id      INTEGER NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    days         INTEGER NOT NULL CHECK (days BETWEEN 1 AND 127),
    start        INTEGER NOT NULL CHECK (start BETWEEN 0 AND 1439),
    "end"        INTEGER NOT NULL CHECK ("end" > start AND "end" <= 1440),
    notify_start INTEGER NOT NULL DEFAULT 0,
    notify_end   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX placements_task ON placements (task_id);

-- +goose Down
DROP TABLE placements;
