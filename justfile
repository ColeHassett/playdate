set dotenv-load

default: dev

fmt:
	go fmt ./src

dev: fmt
	air


