# SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
#
# SPDX-License-Identifier: Apache-2.0

# Build stage
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o hawkeye main.go

# Prep stage for user
FROM alpine:3.23 AS prep
RUN apk add --no-cache ca-certificates
RUN adduser \
    --disabled-password \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid 65532 \
    hawkeye

# Final stage
FROM alpine:3.23
RUN apk add --no-cache iputils
COPY --from=prep /etc/passwd /etc/passwd
COPY --from=prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/hawkeye /hawkeye

USER hawkeye
ENTRYPOINT ["/hawkeye"]