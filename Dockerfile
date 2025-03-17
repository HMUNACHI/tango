FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tango .

FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/tango .
COPY --from=builder /app/config.yaml .

EXPOSE 50051

CMD ["./tango"]
