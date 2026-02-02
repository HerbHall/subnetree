## Tailscale Integration (Plugin)

### Overview

The Tailscale plugin provides automatic device discovery and overlay network connectivity for users running Tailscale. It queries the Tailscale API to enumerate devices on the user's tailnet, enriching NetVantage's device inventory with Tailscale metadata (IP addresses, hostnames, tags, OS, online status). For distributed home labs and multi-site networks, Tailscale eliminates NAT traversal complexity -- devices are reachable by their stable 100.x.y.z addresses regardless of physical location.

**Licensing:** The Tailscale API client library (`tailscale-client-go-v2`) is MIT licensed. No copyleft or licensing conflicts with BSL 1.1.

**Base distribution candidate:** This plugin adds minimal binary size and zero runtime cost when disabled. Flagged for inclusion in the default build if testing confirms no impact on startup time or first-run experience for users without Tailscale.

### Capabilities

| Capability | Description | Phase |
|-----------|-------------|-------|
| Device discovery | Enumerate tailnet devices via Tailscale API (hostname, IPs, OS, tags, last seen, online status) | 2 |
| Tailscale IP enrichment | Add Tailscale IPs (100.x.y.z) to existing device records, enabling monitoring across NAT | 2 |
| Subnet route awareness | Detect Tailscale subnet routers and offer to scan advertised subnets | 2 |
| MagicDNS hostname resolution | Use Tailscale DNS names (e.g., `device.tailnet.ts.net`) for device identification | 2 |
| Scout over Tailscale | Support Scout agent communication via Tailscale IPs (no port forwarding required) | 2 |
| Connectivity preference | Prefer Tailscale IPs for monitoring/remote access when devices are on the tailnet | 3 |
| Tailscale Funnel/Serve guidance | Documentation for exposing NetVantage dashboard via Tailscale Funnel | 1 (docs only) |

### Authentication

| Method | Use Case | Storage |
|--------|----------|---------|
| API key | Personal tailnets, simple setup | Vault-encrypted |
| OAuth client | Organizational tailnets, token refresh | Vault-encrypted (client ID + secret) |

API credentials are stored in the Vault module (encrypted at rest). The plugin never stores credentials outside Vault.

### Device Merging

Tailscale-discovered devices are merged with existing NetVantage device records using a match priority:

1. **MAC address** -- exact match (most reliable)
2. **Hostname** -- case-insensitive match
3. **IP overlap** -- any shared IP between Tailscale device and existing record

If no match is found, a new device record is created with `discovery_method: tailscale`. Merged devices retain their original discovery method and gain Tailscale metadata as supplemental data.

### Dependencies

- **Required:** Vault (for API key / OAuth credential storage)
- **Optional:** Recon (merges Tailscale-discovered devices with scan results)
- **Optional:** Gateway (uses Tailscale IPs for remote access)

### Implementation Notes

- Uses `tailscale-client-go-v2` (MIT) -- lightweight client, no heavy dependency tree
- Respects Tailscale API rate limits (`Retry-After` headers); default sync interval of 5 minutes
- Graceful degradation: if Tailscale API is unreachable, the plugin logs a warning and retries on next sync interval; does not block other discovery methods
- Dashboard shows a "Tailscale" badge on devices discovered or enriched via the tailnet
