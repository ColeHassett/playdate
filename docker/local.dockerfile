# Start by building the application.
FROM golang:1.25 AS builder

WORKDIR /app/
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go install github.com/cosmtrek/air@v1.49.0
RUN go install github.com/a-h/templ/cmd/templ@latest

EXPOSE 8080

CMD ["air"]
