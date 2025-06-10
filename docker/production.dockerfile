# Start by building the application.
FROM golang:1.23 as build

WORKDIR /go/app/
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -C /go/app/ -o /go/app/bin/app

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian11
COPY ./banner.txt /
COPY --from=build /go/app/bin/app /go/app/bin
COPY --from=build /go/app/templates /go/app/templates
CMD ["/go/app/bin"]
