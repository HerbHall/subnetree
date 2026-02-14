# Getting Started with SubNetree

This guide walks you through your first 15 minutes with SubNetree -- from
creating your account to monitoring your first device. No networking experience
required.

## What You Will See

After starting SubNetree and opening [http://localhost:8080](http://localhost:8080)
in your browser, you will see a setup wizard. This only appears on first run.
Once you create your admin account, SubNetree takes you straight to the
dashboard.

## Create Your Account

1. Open [http://localhost:8080](http://localhost:8080) in your browser.
2. The setup wizard asks for a username and password. Pick something you will
   remember -- this is the only account with full access.
3. Click **Create Account**. SubNetree logs you in automatically.

> **Tip:** If you are running SubNetree in Docker with bridge networking
> (`-p 8080:8080`), use `localhost`. If you used `--network host` on Linux,
> use your server's IP address instead.

## Run Your First Scan

SubNetree discovers devices by scanning your local network. Here is how to
start your first scan:

1. From the dashboard, click **Devices** in the sidebar.
2. Click **Scan** at the top of the device list.
3. Enter your network range. For most home networks, this is `192.168.1.0/24`
   or `192.168.0.0/24`. If you are unsure, check your router's settings page
   for the subnet.
4. Click **Start Scan**. A progress bar shows the scan running in real time.
5. Discovered devices appear in the list as they are found.

> **Tip:** A `/24` subnet covers addresses from `.1` to `.254` -- enough for
> most home networks. If you have more devices on a larger range, try
> `192.168.1.0/22` to cover `.0.1` through `.3.254`.

## Explore Discovered Devices

After the scan finishes, your device list shows everything SubNetree found.

![Discovered network devices with search, filtering, and status indicators](../images/device-list.png)

Each device entry shows:

- **Name** -- the hostname if available, or the IP address
- **IP Address** -- where the device lives on your network
- **MAC Address** -- the hardware identifier
- **Manufacturer** -- the company that made the device (identified from the MAC
  address)
- **Status** -- whether the device is currently reachable

Click any device to open its detail page. You will see the full profile
including open ports, operating system (when detectable), and connection
history.

## Set Up Monitoring

Now that you have devices, let SubNetree keep an eye on them. Monitoring checks
run on a schedule and alert you if something goes down.

1. Open any device's detail page by clicking on it.
2. Click **Add Check**.
3. Choose a check type:
   - **ICMP (Ping)** -- sends a ping to the device. Best for "is it up?" monitoring.
   - **TCP** -- connects to a specific port. Good for checking if a service is running.
   - **HTTP** -- requests a web page. Good for web servers and applications.
4. Set the **interval** (how often to check). Every 60 seconds is a good
   starting point.
5. Click **Save**. SubNetree begins monitoring immediately.

> **Tip:** Start with an ICMP check on your most important devices -- your
> router, NAS, or home server. You can always add more checks later.

## View the Topology

The topology page shows a visual map of your network -- which devices are
connected and how they relate to each other.

1. Click **Topology** in the sidebar.
2. You will see a graph with your devices as nodes and their connections as
   lines.
3. Click and drag to rearrange the layout. Scroll to zoom in or out.
4. Click any device node to jump to its detail page.

![Interactive network topology visualization with grid layout](../images/topology-visualization.png)

The topology builds automatically from scan data. As SubNetree learns more
about your network over time, the map becomes more detailed.

## Next Steps

You are up and running. Here are some things to try next:

- **Store credentials** so you can SSH into devices from the browser. See the
  [Common Tasks](common-tasks.md) guide.
- **Set up alerts** to get notified when devices go down. See
  [Common Tasks: Set up alerts](common-tasks.md#set-up-alerts).
- **Install Scout agents** on important servers for deeper monitoring. See
  [Common Tasks: Install Scout on a device](common-tasks.md#install-scout-on-a-device).
- **Something not working?** Check the
  [Troubleshooting](troubleshooting.md) guide.
