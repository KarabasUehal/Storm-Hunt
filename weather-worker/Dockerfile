FROM golang:1.24.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o weather-worker .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/weather-worker .

CMD ["./weather-worker"]