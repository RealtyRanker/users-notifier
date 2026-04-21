FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /users-notifier ./cmd/notifier

# ---

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
RUN mkdir -p /var/log/users-notifier

WORKDIR /app
COPY --from=builder /users-notifier .
COPY config.yaml .

EXPOSE 8080 9091

CMD ["./users-notifier", "-config", "config.yaml"]