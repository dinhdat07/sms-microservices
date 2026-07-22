# Bruno API Test Collection — Server Management Service

> **45 test cases** covering all SMS endpoints: happy path, validation errors,
> auth edge cases, and authorization boundaries.

## Prerequisites

| Tool | Install |
|------|---------|
| Bruno Desktop 3.x | `winget install Bruno.Bruno` or [download](https://www.usebruno.com/downloads) |
| Bruno CLI | `npm install -g @usebruno/cli` |
| Docker 24+ | For infra (postgres, redis, elasticsearch, mailhog) |
| Go 1.26+ | To run the API server and generate test fixtures |

## Quick Start

### 1. Start Infrastructure (NO Kafka)

```bash
cd backend/server-management-service
docker-compose up -d postgres redis elasticsearch mailhog
docker ps --filter "name=sms_" --format "table {{.Names}}\t{{.Status}}"
```

### 2. Start API

```bash
go run ./cmd/api
# Wait for: grpc listening on :50051
```

### 3. Generate Test Fixtures

```bash
cd bruno/testdata
go run generate_fixtures.go
```

### 4. Run Tests

**GUI:** Open Bruno → Open Collection → `bruno/` → select `local` env → Run

**CLI:**
```powershell
.\bruno\scripts\run-smoke.ps1 -Env local
```

## Infrastructure Dependencies

| Service | Port | Used For |
|---------|------|----------|
| PostgreSQL | 5432 | Auth users, server CRUD |
| Redis | 6379 | Server ID cache, distributed lock |
| Elasticsearch | 9200 | Observation logs (reporting reads) |
| MailHog | 1025 | SMTP sink for report emails |

> **Kafka is NOT used.** Reporting runs via internal Goroutine Worker Pool
> per DEVELOPMENT_GUIDE.md.

## Test Case Summary (~45 cases)

| Folder | Cases | Coverage |
|--------|-------|----------|
| `auth/` | 10 | Login (admin/user/wrong pw/invalid email/missing) + Refresh (ok/invalid/empty) + Logout (ok/no token) |
| `servers/` | 27 | Create (8), View (7), Update (4), Delete (3), Import (4), Export (1) |
| `reporting/` | 4 | Request report (ok/no email/invalid date/swapped dates) |
| `health/` | 2 | Health check + OpenAPI spec |
| `authorization/` | 2 | No token (401) + User→Admin endpoint (403) |

## Environment Variables

| Variable | Default |
|----------|---------|
| `baseURL` | `http://localhost:8000` |
| `adminEmail` | `admin@portal.local` |
| `adminPassword` | `Admin@123456` |
| `userEmail` | `user@portal.local` |
| `userPassword` | `User@123456` |
| `testServerName` | `bruno-test-srv` |
| `testServerIPv4` | `192.168.100.1` |
| `testServerIPv4Alt` | `192.168.100.2` |
| `testReportEmail` | `admin@portal.local` |
