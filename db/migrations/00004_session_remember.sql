-- +goose Up
ALTER TABLE sessions ADD COLUMN remember boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE sessions DROP COLUMN remember;
