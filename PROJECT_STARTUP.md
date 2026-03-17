# 🚀 Hawk-Eye Monitoring Stack Startup Guide

This document provides everything you need to get the full monitoring stack (Hawk-Eye, Prometheus, and Grafana) up and running.

## 📋 Table of Contents
1. [Overview](#-overview)
2. [Prerequisites](#-prerequisites)
3. [Quick Start](#-quick-start)
4. [Monitoring Features](#-monitoring-features)
5. [Accessing the Stack](#-accessing-the-stack)
6. [Configuration](#-configuration)
7. [Maintenance & Commands](#-maintenance-commands)

---

## 🔍 Overview
**Hawkeye** is a powerful infrastructure monitoring agent that performs periodic network health checks. This stack includes:
*   **Hawkeye Agent**: The core monitoring engine.
*   **Prometheus**: Scrapes and stores metrics from Hawk-Eye.
*   **Grafana**: Visualizes the data using a pre-configured dashboard.

---

## 🛠 Prerequisites
Ensure you have the following installed:
*   **Go 1.22+**
*   **Docker & Docker Compose**
*   **jq** (optional, for CLI API testing)

---

## 🚀 Quick Start

### 1. Build the Hawk-Eye Binary
Since we are running inside a container (Alpine-based), we need to build the binary for a Linux target:
```bash
CGO_ENABLED=0 GOOS=linux go build -o hawkeye .
```

### 2. Launch the Stack
Start all three services in the background:
```bash
docker compose up -d
```

### 3. Verify Status
Check that all containers are running:
```bash
docker ps
```
You should see `hawkeye`, `prometheus`, and `grafana` in the list.

---

## 📡 Monitoring Features
The stack is configured to test the following features:
*   **Health**: HTTP status checks for external targets.
*   **Latency**: Measures RTT (Round Trip Time) to targets.
*   **DNS**: Validates domain name resolution.
*   **Traceroute**: Tracks the network path to specific IPs or domains.

---

## 🖥 Accessing the Stack

| Service | URL | Description |
| :--- | :--- | :--- |
| **Grafana** | [http://localhost:3000](http://localhost:3000) | **Hawkeye Dashboard** (User: `admin` / Password: `admin`) |
| **Prometheus** | [http://localhost:9090](http://localhost:9090) | Scraper status & Target health |
| **Hawkeye API** | [http://localhost:8080](http://localhost:8080) | Live JSON metrics from the agent |

---

## ⚙️ Configuration

### Updating Checks
You can add or remove monitoring targets by editing the `checks.yaml` file in the project root. The Hawk-Eye agent will automatically pick up changes every **10 seconds**.

**Example `checks.yaml`:**
```yaml
health:
  interval: 10s
  targets: ["https://www.google.com"]
latency:
  interval: 20s
  targets: ["https://www.cloudflare.com"]
```

### Grafana Dashboards
The dashboard is automatically provisioned from `examples/dashboard.json`. To modify it permanently, edit the file or export your changes from the Grafana UI.

---

## 🔧 Maintenance & Commands

**Watch Logs**:
```bash
docker logs -f hawkeye
```

**Stop Entire Stack**:
```bash
docker compose down
```

**Restart Specific Service**:
```bash
docker compose restart hawkeye
```

**Query Local API (Traceroute Example)**:
```bash
curl -s http://localhost:8080/v1/metrics/traceroute | jq .
```
