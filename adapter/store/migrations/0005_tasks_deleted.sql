-- +goose Up
-- a deleted task waits here for its undo. only a person's last delete waits:
-- the next one erases it for good (decisions/0001: nothing is kept).
ALTER TABLE tasks ADD COLUMN deleted INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tasks DROP COLUMN deleted;
