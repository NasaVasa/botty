FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/botty ./cmd/botty

FROM alpine:latest

WORKDIR /app

COPY --from=builder /out/botty ./botty

RUN adduser -D -u 1000 appuser && \
    chown -R appuser:appuser /app
USER appuser

CMD ["./botty"]
