-- +goose Up
-- +goose StatementBegin
CREATE TYPE attendance AS ENUM ('no', 'maybe', 'yes');
CREATE TYPE playdate_status AS ENUM ('pending', 'done');
CREATE TABLE IF NOT EXISTS player (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    discord_id TEXT NOT NULL UNIQUE,
    created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    verification_code VARCHAR(128) NOT NULL
);
CREATE TABLE IF NOT EXISTS playdate (
    id SERIAL PRIMARY KEY,
    game TEXT NOT NULL,
    created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    date TIMESTAMP NOT NULL,
    status playdate_status DEFAULT 'pending' NOT NULL,
    owner_id INT NOT NULL,
    FOREIGN KEY (owner_id) REFERENCES player(id)
);
CREATE TABLE IF NOT EXISTS playdate_player (
    playdate_id INT REFERENCES playdate(id),
    player_id INT REFERENCES  player(id),
    attending attendance DEFAULT 'no' NOT NULL,
    PRIMARY KEY (player_id, playdate_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS playdate_player;
DROP TABLE IF EXISTS playdate;
DROP TABLE IF EXISTS player;
DROP TYPE IF EXISTS attendance CASCADE;
DROP TYPE IF EXISTS playdate_status CASCADE;
-- +goose StatementEnd
