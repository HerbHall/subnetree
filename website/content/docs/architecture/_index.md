---
title: Architecture
weight: 2
---

NetVantage is a modular, self-hosted network monitoring platform built in Go with a React/TypeScript dashboard.

## System Overview

```
                    +------------------+
                    |    Dashboard     |
                    | (React + TS)     |
                    +--------+---------+
                             |
                    REST / WebSocket
                             |
+----------+       +---------+---------+       +----------+
|  Scout   | gRPC  |   NetVantage      |       | Network  |
|  Agent   +------>+   Server (Go)     +------>+ Devices  |
|          |       |                   | ICMP/  | (SNMP,   |
+----------+       | +------+ +------+ | SNMP/  |  mDNS,   |
                   | |Recon | |Pulse | | ARP/   |  UPnP)   |
                   | +------+ +------+ | mDNS   +----------+
                   | |Dsptch | Vault | |
                   | +------+ +------+ |
                   | |Gatway |        |
                   | +------+         |
                   +-------------------+
```

## Components

### NetVantage Server

The core Go application. Hosts the REST API, serves the dashboard SPA, manages the plugin lifecycle, and coordinates all modules.

- **HTTP server** with middleware stack (logging, recovery, CORS, metrics)
- **gRPC server** for Scout agent communication
- **Plugin registry** with dependency resolution and lifecycle management
- **SQLite database** for persistent storage
- **Event bus** for decoupled inter-module communication

### Dashboard

React + TypeScript single-page application served by the Go server. Communicates via REST API and WebSocket for real-time updates.

### Scout Agent

Lightweight agent deployed to endpoints. Connects to the server via gRPC and reports system metrics, health status, and telemetry.

## Plugin Architecture

Every major feature in NetVantage is a plugin. The plugin system is inspired by [Caddy's module architecture](https://caddyserver.com/) (see [ADR-0003](https://github.com/HerbHall/netvantage/blob/main/docs/adr/0003-plugin-architecture-caddy-model.md)).

Plugins implement role interfaces defined in the SDK:

| Role | Interface | Description |
|------|-----------|-------------|
| Scanner | `ScannerPlugin` | Network discovery methods |
| Monitor | `MonitorPlugin` | Health checks and metrics collection |
| Notifier | `NotifierPlugin` | Alert delivery channels |
| Store | `StorePlugin` | Data persistence backends |
| HealthChecker | `HealthChecker` | Component health reporting |

The Plugin SDK is licensed under **Apache 2.0** -- no restrictions on building and distributing plugins.

## Database

NetVantage uses SQLite as its primary database (see [ADR-0002](https://github.com/HerbHall/netvantage/blob/main/docs/adr/0002-sqlite-first-database.md)). Benefits:

- Zero external dependencies -- no separate database server
- Single-file backup and restore
- Sufficient for the target scale (home lab to small business)
- Optional upgrade path to PostgreSQL for larger deployments

## API Design

- RESTful JSON API under `/api/v1/`
- [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807) structured error responses
- `X-NetVantage-Version` header on all responses
- Integer-based protocol versioning (see [ADR-0004](https://github.com/HerbHall/netvantage/blob/main/docs/adr/0004-integer-protocol-versioning.md))

## Licensing Model

Split licensing to balance open-source accessibility with commercial sustainability (see [ADR-0001](https://github.com/HerbHall/netvantage/blob/main/docs/adr/0001-split-licensing-model.md)):

- **Core** (server, agent, built-in modules): BSL 1.1 -- free for personal, home-lab, educational, and non-competing production use. Converts to Apache 2.0 after 4 years.
- **Plugin SDK** (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`, `api/proto/`): Apache 2.0 -- build plugins with no restrictions.

## Further Reading

- [Requirements specifications](https://github.com/HerbHall/netvantage/tree/main/docs/requirements) -- 28 detailed requirement documents
- [Architecture Decision Records](https://github.com/HerbHall/netvantage/tree/main/docs/adr) -- recorded architectural decisions
- [Development setup](/docs/contributing/development-setup) -- get the codebase running locally
