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
COPY cactus-gcp-credentials.json .

COPY --from=builder /app/tango .
EXPOSE 50051

CMD ["./tango"]
