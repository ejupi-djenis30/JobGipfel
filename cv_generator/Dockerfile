# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /cv_generator ./cmd/server

# Runtime stage with Chrome for PDF generation
FROM alpine:3.19

WORKDIR /app

# Install Chrome/Chromium for PDF generation
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont

# Set Chrome path
ENV CHROME_PATH=/usr/bin/chromium-browser

COPY --from=builder /cv_generator .

EXPOSE 8083

ENTRYPOINT ["./cv_generator"]
