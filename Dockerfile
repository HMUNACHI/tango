
FROM golang:1.18 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o tango-server ./src

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /app/tango-server .
COPY --from=builder /app/config.yaml .

EXPOSE 50051
ENTRYPOINT ["./tango-server"]
