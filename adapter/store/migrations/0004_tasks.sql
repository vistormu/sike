-- +goose Up
-- "when" names where a task lives in time, and so its tab (decisions/0005). all
-- four values are allowed here so weekly (m15) and date (m18) need no table
-- rebuild; the service decides which it accepts.
CREATE TABLE tasks (
    id       INTEGER PRIMARY KEY,
    user_id  INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name     TEXT    NOT NULL,
    color    TEXT    NOT NULL,
    "when"   TEXT    NOT NULL CHECK ("when" IN ('weekly', 'date', 'pending', 'someday')),
    position INTEGER NOT NULL, -- ascending; new and moved tasks go on top
    FOREIGN KEY (user_id, color) REFERENCES areas (user_id, color)
);
CREATE INDEX tasks_user_when ON tasks (user_id, "when", position);

-- +goose Down
DROP TABLE tasks;
