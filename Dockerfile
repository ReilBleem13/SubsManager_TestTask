FROM golang:1.26.1-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /app/online_subs ./cmd/app/main.go

FROM alpine:latest AS final

WORKDIR /app

COPY --from=builder /app/online_subs ./online_subs
COPY .env ./
COPY internal/infra/database/migrations ./internal/infra/database/migrations

EXPOSE 8080
CMD ["./online_subs"]