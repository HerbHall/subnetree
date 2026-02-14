# Common Tasks

This guide covers everyday operations you will use after the initial setup.
Each section explains when you would need the task and walks you through it
step by step.

## Add a Device Manually

Sometimes a device does not show up during a scan -- it might be on a
different subnet, behind a firewall, or powered off during the scan. You can
add it by hand.

1. Click **Devices** in the sidebar.
2. Click **Add Device**.
3. Enter the device's IP address and an optional name.
4. Click **Save**.

SubNetree will attempt to reach the device and fill in details like hostname,
MAC address, and manufacturer automatically.

> **Tip:** You can also add devices via the API:
>
> ```bash
> curl -X POST http://localhost:8080/api/v1/recon/devices \
>   -H "Content-Type: application/json" \
>   -H "Authorization: Bearer YOUR_TOKEN" \
>   -d '{"ip_address": "192.168.1.50", "name": "My NAS"}'
> ```

## Create a Monitoring Check

Monitoring checks run on a schedule and tell you whether a device or service is
healthy. SubNetree supports three check types.

1. Open a device's detail page by clicking on it in the device list.
2. Click **Add Check**.
3. Choose the check type:
   - **ICMP (Ping)** -- sends a ping. Use this for basic "is it up?" checks.
     Works for almost any device.
   - **TCP** -- connects to a specific port. Use this to verify a service is
     listening (e.g., port 22 for SSH, port 443 for HTTPS).
   - **HTTP** -- requests a URL and checks the response code. Use this for web
     servers, APIs, or applications with a health endpoint.
4. Set the **interval** -- how often the check runs. 60 seconds is a good
   default. For less critical devices, 5 minutes saves resources.
5. Set the **timeout** -- how long to wait for a response before marking the
   check as failed. 5 seconds works for most home networks.
6. Click **Save**.

The check starts running immediately. Results appear on the device's detail
page and on the monitoring dashboard.

## Set Up Alerts

Alerts notify you when a monitoring check detects a problem. SubNetree tracks
three states: **OK** (healthy), **Warning** (degraded), and **Critical**
(down).

1. Navigate to **Monitoring** in the sidebar.
2. Click **Alerts** to view current alert rules and active alerts.
3. Click **Add Alert Rule**.
4. Choose the check you want to alert on.
5. Set the threshold -- for example, alert when a device fails 3 consecutive
   checks.
6. Click **Save**.

Active alerts appear on the dashboard overview and on the Monitoring page.
SubNetree also uses statistical anomaly detection (EWMA baselines) to flag
unusual behavior automatically.

## Store Credentials in the Vault

The credential vault encrypts and stores login information so you do not have
to re-enter passwords every time you connect to a device.

1. Click **Vault** in the sidebar.
2. If the vault is sealed, enter your vault passphrase to unlock it.
3. Click **Add Credential**.
4. Choose the credential type:
   - **SSH** -- username and password or SSH key for remote shell access
   - **SNMP** -- community string or SNMPv3 credentials for network devices
   - **HTTP Basic** -- username and password for web interfaces
   - Other types as needed
5. Enter the credential details and give it a descriptive name (e.g., "NAS SSH
   root" or "Router admin").
6. Click **Save**.

You can assign stored credentials to devices. When you connect to a device via
SSH or other protocols, SubNetree uses the assigned credential automatically.

> **Tip:** If you set the `SUBNETREE_VAULT_PASSPHRASE` environment variable,
> the vault unlocks automatically when SubNetree starts. See the
> [Troubleshooting](troubleshooting.md#vault-is-sealed-or-prompts-for-a-passphrase)
> guide for details.

## Connect to a Device via SSH

SubNetree includes a web-based SSH terminal powered by the Gateway module. You
can open a shell session to any device directly from your browser -- no
separate SSH client needed.

1. Open the device's detail page.
2. Click **SSH** (or **Connect > SSH**).
3. If the device has an assigned credential, SubNetree uses it automatically.
   Otherwise, you will be prompted for a username and password.
4. A terminal opens in your browser. You have a full shell session to the
   device.
5. Close the tab or click **Disconnect** to end the session.

> **Tip:** Store credentials in the vault first (see above) for one-click
> access without typing passwords.

## Back Up Your Data

Regular backups protect your device inventory, monitoring history, credentials,
and settings.

### Using the CLI

```bash
# Create a backup file
subnetree backup --output /path/to/backup.tar.gz
```

### Using Docker

```bash
# Run the backup command inside the container
docker exec subnetree subnetree backup --output /data/backup.tar.gz

# Copy the backup to your host
docker cp subnetree:/data/backup.tar.gz ./subnetree-backup.tar.gz
```

### Backing Up the Docker Volume Directly

If you prefer, you can back up the entire data volume:

```bash
docker run --rm -v subnetree-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/subnetree-data-backup.tar.gz -C /data .
```

> **Tip:** Schedule backups regularly. A weekly backup is a good starting point
> for most home labs.

## Restore from Backup

If you need to recover your data from a backup:

### Using the CLI

```bash
subnetree restore --input /path/to/backup.tar.gz
```

### Using Docker

```bash
# Copy the backup file into the container
docker cp ./subnetree-backup.tar.gz subnetree:/data/backup.tar.gz

# Run the restore command
docker exec subnetree subnetree restore --input /data/backup.tar.gz

# Restart the container to pick up the restored data
docker restart subnetree
```

> **Tip:** Always stop SubNetree before restoring to avoid conflicts with the
> running database.

## Update SubNetree

New versions include bug fixes, performance improvements, and new features.
Updating takes about a minute.

### Docker (Recommended)

```bash
# Pull the latest image
docker pull ghcr.io/herbhall/subnetree:latest

# Stop and remove the current container
docker stop subnetree
docker rm subnetree

# Start a new container with the updated image
docker run -d --name subnetree -p 8080:8080 -v subnetree-data:/data ghcr.io/herbhall/subnetree:latest
```

Your data is stored in the `subnetree-data` volume, so it persists across
container replacements.

### Docker Compose

```bash
docker compose pull
docker compose up -d
```

### Binary

Download the latest release from the
[GitHub Releases](https://github.com/HerbHall/subnetree/releases) page.
Replace the old binary with the new one and restart SubNetree.

> **Tip:** Back up your data before updating, just in case. See
> [Back up your data](#back-up-your-data) above.

## Install Scout on a Device

The Scout agent runs on devices you want to monitor in depth. It collects
system metrics (CPU, memory, disk, network) and reports back to the SubNetree
server.

### Linux

1. Download the Scout binary from the
   [GitHub Releases](https://github.com/HerbHall/subnetree/releases) page.
   Choose the binary that matches the device's architecture (amd64, arm64,
   etc.).

2. Make it executable:

   ```bash
   chmod +x scout
   ```

3. Start Scout, pointing it at your SubNetree server:

   ```bash
   ./scout --server YOUR_SERVER_IP:9090
   ```

   Replace `YOUR_SERVER_IP` with the IP address of the machine running
   SubNetree.

4. Scout enrolls automatically. You will see the device appear in SubNetree
   with a "Scout" badge within a few seconds.

### Windows

1. Download the Windows Scout binary (`scout.exe`) from the
   [GitHub Releases](https://github.com/HerbHall/subnetree/releases) page.

2. Open a command prompt or PowerShell window and run:

   ```powershell
   .\scout.exe --server YOUR_SERVER_IP:9090
   ```

3. Scout enrolls with the server and begins reporting system metrics.

> **Tip:** For always-on monitoring, set up Scout as a system service. On
> Linux, create a systemd unit file. On Windows, use `sc create` or Task
> Scheduler to start Scout at boot.

## Customize the Dashboard Theme

SubNetree ships with a forest green theme by default, but you can customize
the look of the dashboard.

1. Click the **Settings** icon in the sidebar (or navigate to **Settings >
   Themes**).
2. Browse the available themes.
3. Click a theme to preview it.
4. Click **Apply** to set it as your active theme.

Theme changes take effect immediately. If you do not see the change, do a hard
refresh in your browser (Ctrl+Shift+R on Windows/Linux, Cmd+Shift+R on macOS).

## What is Next

- New to SubNetree? Start with the [Getting Started](getting-started.md)
  walkthrough.
- Running into problems? Check the [Troubleshooting](troubleshooting.md)
  guide.
- Want to access SubNetree remotely over Tailscale? See the
  [Tailscale Deployment](tailscale-deployment.md) guide.
