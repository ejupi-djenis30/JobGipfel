# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /autoapply_service ./cmd/server

# Runtime stage with Chrome for web automation
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont

ENV CHROME_PATH=/usr/bin/chromium-browser

COPY --from=builder /autoapply_service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8084

ENTRYPOINT ["./autoapply_service"]
