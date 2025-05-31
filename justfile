set dotenv-load

default: up


# chain other commands together to avoid typing
up: down fmt
	docker compose -f ./docker/docker-compose.yaml up --build

# I've found that sometimes my containers hang around after quitting an up
down:
	docker compose -f ./docker/docker-compose.yaml down

# basic format incase you editor doesn't
fmt:
	go fmt .

