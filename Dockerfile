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

# Final stage (RHEL Compatible UBI 9)
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

# Install required network utilities and create non-root user
RUN microdnf update -y \
    && microdnf install -y iputils shadow-utils ca-certificates bind-utils libcap \
    && useradd --no-create-home --shell /sbin/nologin --uid 65532 hawkeye \
    && microdnf remove -y shadow-utils \
    && microdnf clean all

COPY --from=builder /app/hawkeye /hawkeye

# Grant raw socket permissions to the Go binary so Traceroute/ICMP works natively without root
RUN setcap cap_net_raw,cap_net_admin+ep /hawkeye

USER hawkeye
ENTRYPOINT ["/hawkeye"]