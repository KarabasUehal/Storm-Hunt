FROM golang:1.24.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o storm-backend main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/storm-backend .

EXPOSE 50051

CMD ["./storm-backend"]