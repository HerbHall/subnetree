# Exposing SubNetree with Tailscale Serve and Funnel

This guide explains how to expose the SubNetree dashboard using Tailscale's
built-in reverse proxy features, without port forwarding or dynamic DNS.

## Serve vs Funnel vs Port Forwarding

| Feature | Tailscale Serve | Tailscale Funnel | Port Forwarding |
| --- | --- | --- | --- |
| **Audience** | Your tailnet only | Public internet | Public internet |
| **HTTPS** | Automatic (LE cert) | Automatic (LE cert) | Manual |
| **Auth required** | Tailscale login | None (app handles it) | None |
| **DNS** | MagicDNS or `*.ts.net` | `*.ts.net` | Manual (DDNS) |
| **Port forwarding** | Not needed | Not needed | Required |
| **Setup** | One command | One command | Router config |

**Recommendation:** Use **Serve** for personal access across your tailnet.
Use **Funnel** only when you need to share the dashboard with someone outside
your tailnet.

## Prerequisites

- Tailscale v1.52+ installed on the SubNetree server host
- SubNetree running on `localhost:8080`
- MagicDNS enabled in your
  [Tailscale admin console](https://login.tailscale.com/admin/dns) (optional
  but recommended)
- For Funnel: HTTPS and Funnel enabled in
  [access controls](https://login.tailscale.com/admin/acls)

## Tailscale Serve (Private -- Tailnet Only)

Serve exposes a local port to other devices on your tailnet over HTTPS with an
automatic TLS certificate.

```bash
# Expose SubNetree's HTTP port on your tailnet
tailscale serve 8080
```

This makes SubNetree available at `https://your-hostname.tailnet-name.ts.net`
for any device on your tailnet.

### Serve with a Custom Port

```bash
# Expose on a different HTTPS port
tailscale serve --https 443 8080
```

### Check Status

```bash
tailscale serve status
```

### Stop Serving

```bash
tailscale serve off
```

## Tailscale Funnel (Public -- Internet)

Funnel exposes a local service to the public internet through Tailscale's relay
infrastructure. Traffic flows through Tailscale's servers, so no ports need to
be opened on your router.

### Enable Funnel in ACLs

Add this to your Tailscale ACL policy (in the admin console):

```jsonc
{
  "nodeAttrs": [
    {
      "target": ["autogroup:member"],
      "attr": ["funnel"]
    }
  ]
}
```

### Start Funnel

```bash
# Expose SubNetree to the public internet
tailscale funnel 8080
```

SubNetree is now reachable at `https://your-hostname.tailnet-name.ts.net` from
anywhere on the internet.

### Check Status

```bash
tailscale funnel status
```

### Stop Funnel

```bash
tailscale funnel off
```

## Docker Setup

When running SubNetree in Docker, the Tailscale client must be on the **host**,
not inside the container:

```bash
# 1. SubNetree container with host networking
docker run -d --name subnetree \
  --network host \
  -v subnetree-data:/data \
  ghcr.io/herbhall/subnetree:latest

# 2. Tailscale Serve on the host (not in the container)
tailscale serve 8080
```

If you run Tailscale inside a container (sidecar pattern), make sure both
containers share the same network namespace.

### Docker Compose with Tailscale Sidecar

For an all-in-one deployment:

```yaml
# docker-compose.yml
services:
  tailscale:
    image: tailscale/tailscale:latest
    hostname: subnetree
    environment:
      - TS_AUTHKEY=tskey-auth-xxxxx  # Generate at admin console
      - TS_STATE_DIR=/var/lib/tailscale
      - TS_SERVE_CONFIG=/config/serve.json
    volumes:
      - tailscale-state:/var/lib/tailscale
      - ./tailscale-config:/config
    cap_add:
      - NET_ADMIN
      - SYS_MODULE
    restart: unless-stopped

  subnetree:
    image: ghcr.io/herbhall/subnetree:latest
    network_mode: "service:tailscale"
    volumes:
      - subnetree-data:/data
    depends_on:
      - tailscale
    restart: unless-stopped

volumes:
  tailscale-state:
  subnetree-data:
```

Create `tailscale-config/serve.json`:

```json
{
  "TCP": {
    "443": {
      "HTTPS": true
    }
  },
  "Web": {
    "your-hostname.tailnet-name.ts.net:443": {
      "Handlers": {
        "/": {
          "Proxy": "http://127.0.0.1:8080"
        }
      }
    }
  }
}
```

Replace `your-hostname.tailnet-name.ts.net` with your actual Tailscale hostname.

## Security Considerations

- **Serve** traffic is restricted to your tailnet -- only authenticated
  Tailscale users can access it
- **Funnel** traffic is public -- SubNetree's built-in JWT authentication
  protects the dashboard, but the login page is exposed
- SubNetree's first-run setup wizard creates an admin account; complete this
  **before** enabling Funnel
- Consider enabling
  [Tailscale SSH](https://tailscale.com/kb/1193/tailscale-ssh) alongside Serve
  for secure management access
- Review your Tailscale ACLs to limit which devices can use Serve and Funnel

## Troubleshooting

**"Funnel not available"** -- Ensure Funnel is enabled in your Tailscale ACL
policy and that your Tailscale client is v1.52 or later.

**"HTTPS certificate error"** -- Tailscale provisions Let's Encrypt certificates
automatically for `*.ts.net` domains. Ensure MagicDNS is enabled in your admin
console.

**Docker: "connection refused"** -- With the sidecar pattern, SubNetree must use
`network_mode: "service:tailscale"` so both containers share the same network
namespace. Verify SubNetree is listening on `0.0.0.0:8080` (not `127.0.0.1`).

**Dashboard loads but API calls fail** -- If using Funnel with SubNetree behind
a proxy, ensure the `Host` header is forwarded correctly. Tailscale Serve/Funnel
handles this automatically.
