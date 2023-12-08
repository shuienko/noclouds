FROM golang:1.21.5-alpine as build

WORKDIR /usr/src/app

COPY src/go.mod src/go.sum ./
RUN go mod download && go mod verify

COPY src/* .
RUN go build -v -o /usr/local/bin/app ./...

FROM alpine:3.19
COPY --from=build /usr/local/bin/app /usr/local/bin/app

ENTRYPOINT ["/usr/local/bin/app"]