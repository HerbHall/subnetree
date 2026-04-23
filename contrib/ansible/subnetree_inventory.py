#!/usr/bin/env python3
"""SubNetree dynamic inventory plugin for Ansible.

Queries the SubNetree API to generate Ansible inventory grouped by
device type, subnet, category, and tags.

Usage:
    # List all hosts and groups
    ansible-inventory -i subnetree_inventory.py --list

    # Get vars for a specific host
    ansible-inventory -i subnetree_inventory.py --host <hostname>

    # Use in a playbook
    ansible-playbook -i subnetree_inventory.py site.yml

Environment variables:
    SUBNETREE_URL       Base URL (default: http://localhost:8080)
    SUBNETREE_USERNAME  Login username
    SUBNETREE_PASSWORD  Login password
    SUBNETREE_TOKEN     Pre-fetched JWT token (skips login)
    SUBNETREE_VERIFY    SSL verification (default: true, set "false" to disable)

Configuration file (optional):
    Place subnetree.yml next to this script or set SUBNETREE_CONFIG:

        url: https://subnetree.local:8080
        username: admin
        password: changeme
        verify_ssl: true
"""

import argparse
import ipaddress
import json
import os
import sys
import urllib.error
import urllib.request

CONFIG_FILENAMES = ["subnetree.yml", "subnetree.yaml", "subnetree.json"]


def load_config():
    """Load configuration from env vars, falling back to config file."""
    config = {
        "url": os.environ.get("SUBNETREE_URL", "http://localhost:8080"),
        "username": os.environ.get("SUBNETREE_USERNAME", ""),
        "password": os.environ.get("SUBNETREE_PASSWORD", ""),
        "token": os.environ.get("SUBNETREE_TOKEN", ""),
        "verify_ssl": os.environ.get("SUBNETREE_VERIFY", "true").lower() != "false",
    }

    config_path = os.environ.get("SUBNETREE_CONFIG", "")
    if not config_path:
        script_dir = os.path.dirname(os.path.abspath(__file__))
        for name in CONFIG_FILENAMES:
            candidate = os.path.join(script_dir, name)
            if os.path.exists(candidate):
                config_path = candidate
                break

    if config_path and os.path.exists(config_path):
        if config_path.endswith(".json"):
            with open(config_path, encoding="utf-8") as f:
                file_config = json.load(f)
        else:
            # Minimal YAML parsing without PyYAML dependency
            file_config = _parse_simple_yaml(config_path)

        for key in ("url", "username", "password", "token"):
            if key in file_config and file_config[key]:
                config[key] = file_config[key]
        if "verify_ssl" in file_config:
            config["verify_ssl"] = file_config["verify_ssl"]

    config["url"] = config["url"].rstrip("/")
    return config


def _parse_simple_yaml(path):
    """Parse a simple key: value YAML file without PyYAML."""
    result = {}
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if ":" in line:
                key, _, value = line.partition(":")
                key = key.strip()
                value = value.strip().strip("'\"")
                if value.lower() == "true":
                    result[key] = True
                elif value.lower() == "false":
                    result[key] = False
                else:
                    result[key] = value
    return result


def _api_request(url, token=None, data=None, method=None):
    """Make an HTTP request to the SubNetree API."""
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    body = json.dumps(data).encode("utf-8") if data else None
    req = urllib.request.Request(url, data=body, headers=headers, method=method)

    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body_text = e.read().decode("utf-8", errors="replace")
        print(
            f"API error: {e.code} {e.reason} - {body_text}",
            file=sys.stderr,
        )
        sys.exit(1)
    except urllib.error.URLError as e:
        print(f"Connection error: {e.reason}", file=sys.stderr)
        sys.exit(1)


def get_token(config):
    """Authenticate and return a JWT token."""
    if config["token"]:
        return config["token"]

    if not config["username"] or not config["password"]:
        print(
            "Error: Set SUBNETREE_USERNAME/SUBNETREE_PASSWORD or SUBNETREE_TOKEN",
            file=sys.stderr,
        )
        sys.exit(1)

    url = f"{config['url']}/api/v1/auth/login"
    result = _api_request(
        url, data={"username": config["username"], "password": config["password"]}
    )
    return result.get("access_token", result.get("token", ""))


def fetch_devices(config, token):
    """Fetch all devices from the SubNetree API with pagination."""
    devices = []
    limit = 200
    offset = 0

    while True:
        url = f"{config['url']}/api/v1/recon/devices?limit={limit}&offset={offset}"
        result = _api_request(url, token=token)

        batch = result.get("devices", [])
        devices.extend(batch)

        total = result.get("total", 0)
        offset += limit
        if offset >= total or not batch:
            break

    return devices


def _subnet_key(ip_str):
    """Return the /24 subnet string for an IP address, or None."""
    try:
        addr = ipaddress.ip_address(ip_str)
        if isinstance(addr, ipaddress.IPv4Address):
            network = ipaddress.ip_network(f"{addr}/24", strict=False)
            return str(network)
    except ValueError:
        pass
    return None


def _sanitize_group(name):
    """Sanitize a string for use as an Ansible group name."""
    return name.replace("-", "_").replace(" ", "_").replace(".", "_").replace("/", "_")


def build_inventory(devices):
    """Build Ansible inventory structure from SubNetree devices."""
    inventory = {
        "_meta": {"hostvars": {}},
        "all": {"children": []},
    }

    groups = {}

    def ensure_group(name):
        safe = _sanitize_group(name)
        if safe not in groups:
            groups[safe] = {"hosts": [], "vars": {}}
        return safe

    for device in devices:
        hostname = device.get("hostname", "")
        ips = device.get("ip_addresses", [])

        if not hostname or not ips:
            continue

        primary_ip = ips[0]
        host_key = hostname

        # Host variables
        hostvars = {
            "ansible_host": primary_ip,
            "subnetree_id": device.get("id", ""),
            "subnetree_device_type": device.get("device_type", "unknown"),
            "subnetree_status": device.get("status", "unknown"),
            "subnetree_mac_address": device.get("mac_address", ""),
            "subnetree_manufacturer": device.get("manufacturer", ""),
            "subnetree_discovery_method": device.get("discovery_method", ""),
            "subnetree_network_layer": device.get("network_layer", 0),
            "subnetree_connection_type": device.get("connection_type", "unknown"),
        }

        if device.get("os"):
            hostvars["subnetree_os"] = device["os"]
        if device.get("location"):
            hostvars["subnetree_location"] = device["location"]
        if device.get("primary_role"):
            hostvars["subnetree_primary_role"] = device["primary_role"]
        if device.get("owner"):
            hostvars["subnetree_owner"] = device["owner"]
        if device.get("notes"):
            hostvars["subnetree_notes"] = device["notes"]
        if len(ips) > 1:
            hostvars["subnetree_ip_addresses"] = ips
        if device.get("tags"):
            hostvars["subnetree_tags"] = device["tags"]

        custom = device.get("custom_fields") or {}
        for k, v in custom.items():
            hostvars[f"subnetree_custom_{_sanitize_group(k)}"] = v

        inventory["_meta"]["hostvars"][host_key] = hostvars

        # Group by device type
        dtype = device.get("device_type", "unknown")
        group = ensure_group(f"type_{dtype}")
        groups[group]["hosts"].append(host_key)

        # Group by status
        status = device.get("status", "unknown")
        group = ensure_group(f"status_{status}")
        groups[group]["hosts"].append(host_key)

        # Group by subnet
        subnet = _subnet_key(primary_ip)
        if subnet:
            group = ensure_group(f"subnet_{subnet}")
            groups[group]["hosts"].append(host_key)

        # Group by category
        if device.get("category"):
            group = ensure_group(f"category_{device['category']}")
            groups[group]["hosts"].append(host_key)

        # Group by owner
        if device.get("owner"):
            group = ensure_group(f"owner_{device['owner']}")
            groups[group]["hosts"].append(host_key)

        # Group by tags
        for tag in device.get("tags") or []:
            group = ensure_group(f"tag_{tag}")
            groups[group]["hosts"].append(host_key)

        # Group by network layer
        layer = device.get("network_layer", 0)
        layer_names = {
            1: "layer_gateway",
            2: "layer_distribution",
            3: "layer_access",
            4: "layer_endpoint",
        }
        if layer in layer_names:
            group = ensure_group(layer_names[layer])
            groups[group]["hosts"].append(host_key)

    # Add groups to inventory
    for name, data in sorted(groups.items()):
        inventory[name] = data
        inventory["all"]["children"].append(name)

    return inventory


def get_host_vars(devices, hostname):
    """Get variables for a specific host."""
    inventory = build_inventory(devices)
    return inventory["_meta"]["hostvars"].get(hostname, {})


def main():
    parser = argparse.ArgumentParser(description="SubNetree Ansible dynamic inventory")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--list", action="store_true", help="List all hosts and groups")
    group.add_argument("--host", help="Get variables for a specific host")
    args = parser.parse_args()

    config = load_config()
    token = get_token(config)
    devices = fetch_devices(config, token)

    if args.list:
        inventory = build_inventory(devices)
        print(json.dumps(inventory, indent=2))
    elif args.host:
        hostvars = get_host_vars(devices, args.host)
        print(json.dumps(hostvars, indent=2))


if __name__ == "__main__":
    main()
