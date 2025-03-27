CREATE TYPE status AS ENUM ('no', 'maybe', 'yes');

CREATE TABLE IF NOT EXISTS playdate (
    id SERIAL PRIMARY KEY,
    game TEXT NOT NULL,
    created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    date TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS player (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playdate_player (
    player_id INT REFERENCES  player(id),
    playdate_id INT REFERENCES playdate(id),
    attending STATUS DEFAULT 'no' NOT NULL,
    PRIMARY KEY (player_id, playdate_id)
);
