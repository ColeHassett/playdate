# Start by building the application.
FROM golang:1.25 AS builder

RUN go install github.com/cosmtrek/air@latest
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app/
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air"]
