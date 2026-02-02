---
title: Introducing NetVantage
date: 2025-02-02
authors:
  - name: Herb Hall
    link: https://github.com/HerbHall
tags:
  - announcement
  - roadmap
---

NetVantage is a modular, self-hosted network monitoring and management platform -- and today we're making it public.

<!--more-->

## The Problem

If you manage a home lab, a small business network, or client infrastructure as an MSP, you've probably assembled a patchwork of tools: one for device discovery, another for monitoring, a separate solution for remote access, and yet another for credential management. Each has its own setup, its own update cycle, and its own learning curve.

We set out to build something better.

## What NetVantage Is

NetVantage combines five capabilities into a single self-hosted application:

- **Recon** -- Network scanning and device discovery (ARP, ICMP, SNMP, mDNS, UPnP)
- **Pulse** -- Health monitoring, metrics collection, and alerting
- **Dispatch** -- Scout agent enrollment and management across Windows, Linux, and macOS
- **Vault** -- Encrypted credential storage with zero-knowledge architecture
- **Gateway** -- Browser-based remote access via SSH, RDP, and HTTP proxy

Every module is a plugin. You can replace, extend, or supplement any part of the system. The Plugin SDK is licensed under Apache 2.0, so there are no restrictions on building integrations.

## Design Principles

Four principles guide every decision:

1. **Ease of use first.** You shouldn't need a tech degree to understand your network health.
2. **Sensible defaults, deep customization.** Install and go. Then make it yours.
3. **Stability and security are non-negotiable.** If a feature compromises either, it doesn't ship.
4. **Time to First Value under 10 minutes.** Download, install, see your network.

## Current Status

NetVantage is in **Phase 1: Foundation**. Here's what's implemented:

- Go server with HTTP API and middleware stack
- Plugin registry with lifecycle management and dependency resolution
- SQLite database with repository pattern
- Health and plugin status endpoints
- JWT authentication framework
- Prometheus metrics integration
- CI/CD pipeline across Windows, Linux, and macOS

## What's Next

The immediate roadmap:

- **Phase 1 (continued)**: React dashboard shell, agentless network scanning, topology visualization
- **Phase 1b**: Windows Scout agent with gRPC communication
- **Phase 2**: SNMP/mDNS/UPnP discovery, comprehensive monitoring, alerting, multi-tenancy
- **Phase 3**: Browser-based remote access, encrypted credential vault
- **Phase 4**: MQTT/IoT awareness, cross-platform agents, ecosystem growth

## Licensing

NetVantage uses a split licensing model:

- **Core**: BSL 1.1 -- free for personal, home-lab, educational, and non-competing production use. Converts to Apache 2.0 after 4 years.
- **Plugin SDK**: Apache 2.0 -- no restrictions.

This means individual users and small businesses can use NetVantage freely, while the project maintains a sustainable path for commercial development.

## Get Involved

- **Star the repo**: [github.com/HerbHall/netvantage](https://github.com/HerbHall/netvantage)
- **Report issues**: [Issue tracker](https://github.com/HerbHall/netvantage/issues)
- **Contribute**: [Contributing guide](/docs/contributing)
- **Support development**: [GitHub Sponsors](https://github.com/sponsors/HerbHall) | [Ko-fi](https://ko-fi.com/herbhall) | [Buy Me a Coffee](https://buymeacoffee.com/herbhall)

We're building NetVantage in the open and welcome feedback at every stage. If you manage networks and want a unified, self-hosted solution, we'd love to hear from you.
