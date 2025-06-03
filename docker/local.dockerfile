# Start by building the application.
FROM golang:1.23 as builder

WORKDIR /app/
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go install github.com/cosmtrek/air@v1.49.0

EXPOSE 8080

CMD ["air"]
