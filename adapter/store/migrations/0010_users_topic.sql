-- +goose Up
-- a person's ntfy topic: where their notifications go. '' is none: no
-- notifications.
ALTER TABLE users ADD COLUMN topic TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN topic;
