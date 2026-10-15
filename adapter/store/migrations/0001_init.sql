-- +goose Up
-- the baseline. tables arrive with the milestone that needs them (users: m5).
SELECT 1;

-- +goose Down
SELECT 1;
