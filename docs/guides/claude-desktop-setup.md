# Claude Desktop Integration

This guide shows you how to connect Claude Desktop to SubNetree's MCP server, giving Claude access to your device inventory and hardware profiles directly.

## Prerequisites

- SubNetree server running (default http://localhost:8080)
- [Claude Desktop](https://claude.ai/download) installed
- SubNetree MCP API key configured (optional, for authenticated access)

## Configuration

To add SubNetree to Claude Desktop, edit the `claude_desktop_config.json` file:

### On macOS/Linux

```bash
$HOME/.config/Claude/claude_desktop_config.json
```

### On Windows

```
%APPDATA%\Claude\claude_desktop_config.json
```

Add or update the `mcpServers` section:

```json
{
  "mcpServers": {
    "subnetree": {
      "command": "/path/to/subnetree",
      "args": ["mcp"],
      "env": {}
    }
  }
}
```

### Setting the correct path

- **Linux/macOS**: Full path to the binary, e.g. `/usr/local/bin/subnetree` or `$HOME/.local/bin/subnetree`
- **Windows**: Full path with forward slashes, e.g. `C:/Program Files/SubNetree/subnetree.exe`
- **macOS (Homebrew)**: `/usr/local/bin/subnetree`
- **Docker**: If SubNetree runs in a container, mount the binary or use the HTTP transport (see below)

The `mcp` subcommand starts the MCP server in stdio mode, communicating over stdin/stdout with Claude Desktop.

## Available Tools

Claude Desktop can now use these seven tools to query your SubNetree instance:

1. **get_device** -- Retrieve full details for a specific device by ID
2. **list_devices** -- Get a paginated list of all discovered devices (with pagination support)
3. **get_hardware_profile** -- View hardware details (CPU, memory, storage, GPU) for a device
4. **get_fleet_summary** -- Aggregate statistics across your entire fleet (total devices, OS distribution, resource totals)
5. **query_devices** -- Search devices by hardware criteria (OS, CPU cores, memory, storage)
6. **get_stale_devices** -- Find devices that haven't checked in recently (customizable threshold)
7. **get_service_inventory** -- List services and applications running on devices

## Example Queries

Ask Claude questions like:

- "Show me all devices running Ubuntu"
- "How many devices have more than 16GB of RAM?"
- "Which devices haven't checked in for over 24 hours?"
- "What's the hardware summary of my fleet?"
- "Give me a breakdown of operating systems in my network"
- "List all services running on my servers"

Claude uses SubNetree's MCP tools to answer these questions directly from your live device data.

## Troubleshooting

### Connection Refused

**Error**: "Connection to subnetree MCP server failed"

**Solution**: Verify that:
- SubNetree server is running (`http://localhost:8080` responds)
- The binary path in `claude_desktop_config.json` is correct
- On macOS/Linux, the binary has execute permissions: `chmod +x /path/to/subnetree`

### Binary Not Found

**Error**: "Cannot execute /path/to/subnetree: file not found"

**Solution**: Verify the path exists and is spelled correctly. Use the full absolute path (not `~/`). For example:
- Correct: `/Users/john/bin/subnetree`
- Incorrect: `~/bin/subnetree`

### No Devices Returned

**Error**: Tool returns "no devices found" even though you have devices

**Solution**: You need to run at least one network scan first:
1. Open http://localhost:8080 in your browser
2. Go to **Devices** → **Scan**
3. Enter your network range (e.g., `192.168.1.0/24`)
4. Click **Start Scan** and wait for it to complete

Claude will then see the discovered devices.

### Authentication Error

**Error**: "401 Unauthorized" when Claude tries to query devices

**Solution**: If your SubNetree server requires API key authentication, you must use the HTTP transport instead of stdio. See "HTTP Transport" section below.

## HTTP Transport (Remote or Authenticated Access)

For remote servers or when API key authentication is enabled, use HTTP transport instead of stdio:

```json
{
  "mcpServers": {
    "subnetree": {
      "url": "http://your-server:8080/api/v1/mcp/",
      "headers": {
        "Authorization": "Bearer your-api-key-here"
      }
    }
  }
}
```

Replace:
- `your-server` with the IP or hostname of your SubNetree instance
- `your-api-key-here` with the actual API key from your SubNetree configuration

This allows Claude Desktop to connect to SubNetree over the network with authentication.
