FROM golang:1.22.5-alpine AS build

WORKDIR /usr/src/app

# Copy go.mod and go.sum for dependency management
COPY src/go.mod src/go.sum ./
RUN go mod download && go mod verify

# Copy the entire source directory structure (not just files)
COPY src/ ./

# Build the main package only
RUN go build -v -o /usr/local/bin/app .

FROM alpine:3.20.1
COPY --from=build /usr/local/bin/app /usr/local/bin/app

HEALTHCHECK --interval=300s --timeout=10s --start-period=10s --retries=2 CMD [ -f /state.txt ] || exit 1

ENTRYPOINT ["/usr/local/bin/app"]