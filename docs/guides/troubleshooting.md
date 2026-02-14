# Troubleshooting

This guide covers the most common issues you may run into with SubNetree and
how to fix them. Each section describes the symptom, explains what is happening,
and gives you step-by-step fixes.

## SubNetree will not start

The server fails to start, or the Docker container exits immediately after
launching.

**Common causes:**

- Another application is already using port 8080 or 9090.
- The data directory or Docker volume has incorrect permissions.
- The Docker image failed to download completely.

**Fixes:**

1. Check if another application is using port 8080:

   ```bash
   # Linux / macOS
   lsof -i :8080

   # Windows (PowerShell)
   netstat -ano | findstr :8080
   ```

2. If the port is in use, either stop the other application or change
   SubNetree's port:

   ```bash
   docker run -d --name subnetree -p 9080:8080 -v subnetree-data:/data ghcr.io/herbhall/subnetree:latest
   ```

   Then open [http://localhost:9080](http://localhost:9080) instead.

3. Check Docker logs for error messages:

   ```bash
   docker logs subnetree
   ```

4. If you see a database or permission error, remove the volume and start
   fresh:

   ```bash
   docker rm -f subnetree
   docker volume rm subnetree-data
   docker run -d --name subnetree -p 8080:8080 -v subnetree-data:/data ghcr.io/herbhall/subnetree:latest
   ```

## No devices found after scanning

You ran a scan but the device list is empty or shows far fewer devices than
expected.

**Common causes:**

- The subnet range does not match your actual network.
- Docker bridge networking isolates the container from your LAN.
- A firewall is blocking ICMP (ping) traffic.

**Fixes:**

1. Verify your network range. On most home networks, your computer's IP looks
   like `192.168.1.x` or `192.168.0.x`. The scan range should match -- for
   example, `192.168.1.0/24`.

2. Check your computer's IP to confirm the correct subnet:

   ```bash
   # Linux / macOS
   ip addr show | grep "inet "

   # Windows (PowerShell)
   ipconfig | findstr "IPv4"
   ```

3. If you are using Docker with bridge networking (`-p 8080:8080`), the
   container sees a Docker-internal network, not your LAN. For full discovery,
   use host networking on Linux:

   ```bash
   docker run -d --name subnetree --network host -v subnetree-data:/data ghcr.io/herbhall/subnetree:latest
   ```

   > **Note:** Host networking is not available on Docker Desktop for macOS or
   > Windows. On those platforms, bridge networking works but may discover
   > fewer devices.

4. If your firewall blocks ICMP, some devices will not respond to pings. Check
   your host firewall settings and allow outbound ICMP traffic.

## Dashboard shows "connection refused"

You can run SubNetree but your browser shows a "connection refused" error when
you try to open the dashboard.

**Common causes:**

- The Docker port mapping is missing or incorrect.
- SubNetree is bound to a different address than you are using.
- The container has not finished starting yet.

**Fixes:**

1. Verify the container is running:

   ```bash
   docker ps --filter name=subnetree
   ```

   If it is not listed, check `docker ps -a` to see if it exited.

2. Make sure you included `-p 8080:8080` when starting the container. Without
   this flag, Docker does not expose the port to your browser.

3. Wait a few seconds after starting the container. SubNetree needs a moment
   to initialize the database and start listening.

4. If you mapped to a different port (e.g., `-p 9080:8080`), use that port in
   your browser: [http://localhost:9080](http://localhost:9080).

## Devices show as "offline" but are reachable

You can ping a device from your computer, but SubNetree shows it as offline or
in a warning state.

**Common causes:**

- The monitoring check has not run yet (check interval has not elapsed).
- The device's firewall blocks ICMP from the Docker container's IP.
- The check type does not match the device's capabilities.

**Fixes:**

1. Check the monitoring interval. If you set checks to run every 5 minutes,
   wait at least one full interval before expecting results.

2. Verify the check type matches what the device supports:
   - Use **ICMP** for general reachability (most devices respond to pings).
   - Use **TCP** if the device blocks pings but runs a service on a known port
     (e.g., port 22 for SSH, port 80 for HTTP).
   - Use **HTTP** for web servers or applications with a health endpoint.

3. If you are running Docker with bridge networking, the container pings from a
   Docker-internal IP. Some devices or firewalls may not respond to traffic
   from that range. Switching to host networking resolves this on Linux.

## Scout agent will not connect

You installed the Scout agent on a remote device but it does not appear in
SubNetree.

**Common causes:**

- The gRPC port (9090) is not reachable from the agent's device.
- The agent configuration points to the wrong server address.
- A firewall is blocking traffic on port 9090.

**Fixes:**

1. Verify the Scout agent can reach the server's gRPC port:

   ```bash
   # From the device running Scout
   curl -v telnet://YOUR_SERVER_IP:9090
   ```

   Replace `YOUR_SERVER_IP` with the actual IP address of your SubNetree
   server.

2. Check the agent's configuration. The `--server` flag should point to your
   server's IP and gRPC port:

   ```bash
   ./scout --server 192.168.1.10:9090
   ```

3. If you are using Docker with bridge networking for SubNetree, you need to
   also expose the gRPC port:

   ```bash
   docker run -d --name subnetree -p 8080:8080 -p 9090:9090 -v subnetree-data:/data ghcr.io/herbhall/subnetree:latest
   ```

4. Check your server's firewall to make sure port 9090 is open for inbound
   TCP traffic.

## Vault is sealed or prompts for a passphrase

SubNetree starts but the credential vault is sealed. Stored credentials are not
accessible until the vault is unlocked.

**What is happening:** The vault encrypts all stored credentials with a
passphrase. When SubNetree starts, the vault is sealed by default for security.
The server still runs normally -- you just cannot access stored credentials
until the vault is unlocked.

**Fixes:**

1. Unlock the vault through the dashboard: go to **Settings > Vault** and
   enter your vault passphrase.

2. To auto-unlock the vault on startup, set the passphrase as an environment
   variable:

   ```bash
   docker run -d --name subnetree \
     -p 8080:8080 \
     -e SUBNETREE_VAULT_PASSPHRASE=your-passphrase-here \
     -v subnetree-data:/data \
     ghcr.io/herbhall/subnetree:latest
   ```

   > **Tip:** For Docker Compose, add `SUBNETREE_VAULT_PASSPHRASE` to the
   > `environment:` section of your compose file.

3. If you forgot your vault passphrase, there is no way to recover stored
   credentials. You will need to delete the vault data and re-enter your
   credentials. The rest of your SubNetree data (devices, checks, settings)
   is unaffected.

## High CPU or memory usage

SubNetree is using more system resources than expected.

**Common causes:**

- Too many monitoring checks running at short intervals.
- Large subnets being scanned frequently.
- Many devices being monitored simultaneously.

**Fixes:**

1. Increase check intervals. If you have 100 devices pinged every 10 seconds,
   that is 10 pings per second. Changing the interval to 60 seconds reduces
   load significantly.

2. Reduce the scan frequency. Scans are CPU-intensive while running. Schedule
   them less often (e.g., once per hour instead of every 5 minutes).

3. Check the number of active monitoring checks in **Monitoring > Checks**.
   Consider removing checks for devices that do not need active monitoring.

4. SubNetree is designed to run comfortably on modest hardware (a Raspberry Pi
   4 can monitor 50+ devices). If you are exceeding your hardware's capacity,
   prioritize which devices need active monitoring.

## Theme changes do not apply

You changed the dashboard theme in Settings but the interface still looks the
same.

**Fixes:**

1. Clear your browser cache and do a hard refresh:
   - **Windows / Linux:** press Ctrl+Shift+R
   - **macOS:** press Cmd+Shift+R

2. Try opening the dashboard in a private or incognito window. If the new
   theme appears there, your browser is caching the old stylesheet.

3. If using multiple browser tabs, close all SubNetree tabs and open a fresh
   one.

## Still stuck?

If none of the above solutions help:

1. Check the server logs for detailed error messages:

   ```bash
   docker logs subnetree --tail 100
   ```

2. Search the [GitHub Issues](https://github.com/HerbHall/subnetree/issues)
   to see if someone else has reported the same problem.

3. Open a new issue with your SubNetree version, operating system, Docker
   version, and the relevant log output. The more detail you provide, the
   faster the community can help.
