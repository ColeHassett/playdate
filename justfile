set dotenv-load

default: dev

fmt:
	go fmt .

dev: fmt
	air
