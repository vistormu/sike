-- +goose Up
-- one area per color per person: the color is the key, the name is the person's.
CREATE TABLE areas (
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    color   TEXT    NOT NULL CHECK (color IN ('lavender', 'sky', 'apricot', 'coral', 'mint', 'lime', 'pink')),
    name    TEXT    NOT NULL,
    PRIMARY KEY (user_id, color)
);
CREATE UNIQUE INDEX areas_user_name ON areas (user_id, name COLLATE NOCASE);

-- people made before areas existed get the defaults (model.DefaultAreas)
INSERT INTO areas (user_id, color, name)
SELECT u.id, d.column1, d.column2
FROM users u
CROSS JOIN (VALUES ('lavender', 'Self'), ('sky', 'Work'), ('apricot', 'Body'), ('coral', 'Admin'),
                   ('mint', 'Home'), ('lime', 'Social'), ('pink', 'Fun')) d;

-- +goose Down
DROP TABLE areas;
