# --------------------------------------------
# Dockerfile for the Tango server.
# This multi-stage build compiles the Go binary using the golang:1.20 builder,
# then packages it in an Alpine Linux runtime image.
# An entrypoint script is used to validate required environment variables.
# --------------------------------------------
FROM golang:1.20 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o tango .

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY config.yaml .

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY --from=builder /app/tango .
EXPOSE 50051

ENTRYPOINT ["/entrypoint.sh"]
CMD ["./tango"]
