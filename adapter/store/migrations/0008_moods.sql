-- +goose Up
-- a person's mood on a day: 1–5 (model.Faces). one per day; none is no row.
CREATE TABLE moods (
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    date    TEXT    NOT NULL,
    mood    INTEGER NOT NULL CHECK (mood BETWEEN 1 AND 5),
    PRIMARY KEY (user_id, date)
);

-- +goose Down
DROP TABLE moods;
