-- +goose Up
CREATE TABLE users (
    id    INTEGER PRIMARY KEY,
    name  TEXT NOT NULL UNIQUE COLLATE NOCASE,
    emoji TEXT NOT NULL
);

-- +goose Down
DROP TABLE users;
