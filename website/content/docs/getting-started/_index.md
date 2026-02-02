---
title: Getting Started
weight: 1
---

Get NetVantage running in under 10 minutes.

## Prerequisites

- **Go 1.25+** (for building from source)
- **Make** (optional but recommended)
- **Git** (for cloning the repository)

## Build from Source

```bash
git clone https://github.com/HerbHall/netvantage.git
cd netvantage
make build
```

This produces two binaries in `bin/`:
- `netvantage` -- the server
- `scout` -- the agent

## Run the Server

```bash
./bin/netvantage
```

Or with a custom configuration file:

```bash
./bin/netvantage -config configs/netvantage.example.yaml
```

The server starts on `http://localhost:8080` by default.

## Verify It's Running

```bash
curl http://localhost:8080/api/v1/health
```

You should see a JSON response with system health status.

## Run the Scout Agent

On remote machines you want to monitor:

```bash
./bin/scout -server localhost:9090 -interval 30
```

The agent connects to the server via gRPC and reports metrics at the configured interval.

## What's Next?

- [Installation options](installation) -- binary, Docker, Docker Compose
- [Configuration reference](configuration) -- all YAML keys, environment variables, and defaults
- [Architecture overview](/docs/architecture) -- how the system fits together
