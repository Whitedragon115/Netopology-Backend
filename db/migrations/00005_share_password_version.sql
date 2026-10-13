-- +goose Up
ALTER TABLE share_links ADD COLUMN password_version bigint NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE share_links DROP COLUMN password_version;
