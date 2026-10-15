-- +goose Up
-- what the notifier sent, so a restart or a late tick never sends it twice.
-- key names the moment ("start:p3:2026-10-14T18:00"); rows older than a day
-- are dropped: this is not a history (decisions/0001).
CREATE TABLE notifications_sent (
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    key     TEXT    NOT NULL,
    sent_at TEXT    NOT NULL,
    PRIMARY KEY (user_id, key)
);

-- +goose Down
DROP TABLE notifications_sent;
