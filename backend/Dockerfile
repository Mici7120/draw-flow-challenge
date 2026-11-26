FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM python:3.12-alpine

RUN adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /server /app/server

USER appuser
EXPOSE 8081
ENTRYPOINT ["/app/server"]
