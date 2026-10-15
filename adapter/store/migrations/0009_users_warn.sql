-- +goose Up
-- a person's deadline warning, in minutes: 1 hour, 1 day (the default), 2 days.
ALTER TABLE users ADD COLUMN warn INTEGER NOT NULL DEFAULT 1440 CHECK (warn IN (60, 1440, 2880));

-- +goose Down
ALTER TABLE users DROP COLUMN warn;
