FROM golang:1.18 AS builder
WORKDIR /app

ENV ENV=production

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o tango ./src

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /app/tango .
COPY --from=builder /app/config.yaml .

EXPOSE 50051
ENTRYPOINT ["./tango"]
