---
phase: delivery
updated: 2026-03-08T01:00:00Z
updated_by: claude-code
---

# SubNetree -- Current State

## Phase

v0.6.2 shipped (2026-03-07): WiFi scanning, Copilot integration, release gate.
Pre-launch: deployment validation phases pending before community launch.

## What Is Running

- Docker images: GHCR `ghcr.io/herbhall/subnetree:latest` and `:nightly`
- CI: GitHub Actions (build, test, lint, CodeQL, nightly builds)

## In Flight

- #528: Copilot auto-review setup

## Queued

- #493-#498: deployment validation (Docker Desktop, UNRAID, Proxmox)
- #499: content capture for community launch
- #489: Ansible dynamic inventory plugin
- #487: community engagement launch prep

## Last Session Summary

Shipped v0.6.2 with 40+ features. Added Copilot integration, release
gate workflows, nightly builds, and DevKit conformance updates.

## Start Here (Cold Start Protocol)

1. Read this file
2. Call `samverk get_digest --since 168h` if MCP is configured
3. Read open issues if relevant to the task
4. Proceed -- do not ask the user to explain project state
