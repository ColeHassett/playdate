-- +goose Up
-- +goose StatementBegin
ALTER TABLE player ADD COLUMN session_id VARCHAR(128) NOT NULL;
ALTER TABLE player ADD COLUMN oauth_token VARCHAR(128);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE player DROP COLUMN session_id;
ALTER TABLE player DROP COLUMN oauth_token;
-- +goose StatementEnd
