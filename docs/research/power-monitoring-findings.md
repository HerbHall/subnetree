# Power Monitoring and Energy Cost Tracking Research

## Executive Summary

Power consumption and energy cost tracking is a high-ROI pain point affecting 70%+ of the
homelab community, with no simple, self-hosted solution available today. This document covers
MQTT smart plug protocols, SNMP PDU MIBs, electricity rate APIs, a proposed SubNetree plugin
architecture, and dashboard UX design. The recommended approach is a phased implementation
starting with MQTT smart plug ingestion and manual rate configuration, expanding to SNMP PDU
polling and API-based rate lookups in later phases.

## 1. MQTT Smart Plug Protocols

### 1.1 Shelly Gen2 (Native MQTT)

Shelly Gen2 devices (Plus/Pro series) have built-in MQTT support with power metering on
applicable models (e.g., Shelly Plus Plug S, Shelly Pro 4PM).

**Discovery:** mDNS (`_shelly._tcp`) + MQTT status announcements on connect.

**MQTT Topic Structure:**

- Status published to: `<topic_prefix>/status/switch:<id>`
- Commands accepted on: `<topic_prefix>/command/switch:<id>`
- Errors published to: `<topic_prefix>/error/switch:<id>`

The `<topic_prefix>` defaults to `<device_id>` (e.g., `shellyplusplugs-aabbccddeeff`) but is
user-configurable.

**Status Payload (Switch with Power Metering):**

```json
{
  "id": 0,
  "source": "init",
  "output": true,
  "apower": 142.7,
  "voltage": 120.3,
  "current": 1.19,
  "pf": 0.99,
  "freq": 60.0,
  "aenergy": {
    "total": 1587.234,
    "by_minute": [2381.109, 2374.870, 2389.451],
    "minute_ts": 1706745600
  },
  "temperature": {
    "tC": 38.2,
    "tF": 100.8
  }
}
```

**Key Fields:**

| Field | Type | Unit | Description |
|-------|------|------|-------------|
| `apower` | number | Watts | Instantaneous active power |
| `voltage` | number | Volts | Supply voltage |
| `current` | number | Amps | Load current |
| `pf` | number | ratio | Power factor (0-1) |
| `freq` | number | Hz | Line frequency |
| `aenergy.total` | number | Wh | Cumulative energy consumed |
| `aenergy.by_minute` | number[] | mWh | Last 3 complete minutes of energy flow |
| `temperature.tC` | number | C | Device temperature |

**Update Frequency:** Status is published on change (configurable threshold) or on explicit
`status_update` command. The `aenergy.by_minute` array provides per-minute granularity when
the device clock is synced.

**Commands:** `on`, `off`, `toggle`, `status_update` (triggers status republish).

**Authentication:** Shelly Gen2 MQTT supports username/password authentication configured via
device web UI or RPC. No per-topic ACLs; relies on broker-level security.

### 1.2 TP-Link Tapo P110 (via Zigbee2MQTT)

The Tapo P110 is a Wi-Fi smart plug with energy monitoring. It does not natively support MQTT
but can be bridged via Zigbee2MQTT when using the Zigbee-enabled variant (P110 with Matter
support) or through Home Assistant integrations. The more common path for homelab users is
the Zigbee version exposed through Zigbee2MQTT.

**MQTT Topic:** `zigbee2mqtt/<friendly_name>`

**Payload Format (Zigbee2MQTT Exposes):**

```json
{
  "state": "ON",
  "power": 85.3,
  "voltage": 121.5,
  "current": 0.71,
  "energy": 12.45,
  "linkquality": 156
}
```

**Key Fields:**

| Field | Type | Unit | Description |
|-------|------|------|-------------|
| `power` | number | Watts | Instantaneous power consumption |
| `voltage` | number | Volts | Supply voltage |
| `current` | number | Amps | Load current |
| `energy` | number | kWh | Cumulative energy consumed |
| `linkquality` | number | LQI | Zigbee link quality indicator |

**Update Frequency:** Configurable reporting interval in Zigbee2MQTT, typically 10-60 seconds
for power attributes. Energy updates depend on the device firmware (usually every 60 seconds).

### 1.3 Zigbee2MQTT Generic Power Monitoring

Zigbee2MQTT provides a standard MQTT interface for all supported Zigbee power monitoring
devices. The topic structure and payload format are consistent across vendors.

**Topic Pattern:**

- State: `zigbee2mqtt/<friendly_name>` (JSON payload)
- Set: `zigbee2mqtt/<friendly_name>/set` (JSON or key-value)
- Get: `zigbee2mqtt/<friendly_name>/get`
- Availability: `zigbee2mqtt/<friendly_name>/availability`
- Bridge info: `zigbee2mqtt/bridge/info`
- Device list: `zigbee2mqtt/bridge/devices`

**Standard Power Monitoring Fields (Exposes):**

All Zigbee power monitoring devices expose a consistent set of fields:

| Field | Type | Unit | Zigbee Cluster |
|-------|------|------|----------------|
| `power` | number | W | Electrical Measurement (0x0B04) |
| `voltage` | number | V | Electrical Measurement (0x0B04) |
| `current` | number | A | Electrical Measurement (0x0B04) |
| `energy` | number | kWh | Metering (0x0702) |

**Supported Devices (common in homelabs):**

- Sonoff S31 Lite ZB
- Innr SP 224
- IKEA TRADFRI control outlet
- BlitzWolf BW-SHP13 / BW-SHP15
- Tuya TS011F variants

**Device Discovery:** Zigbee2MQTT publishes a retained device list to `zigbee2mqtt/bridge/devices`
containing IEEE addresses, model IDs, manufacturer info, and capabilities (exposes). SubNetree
can parse this to auto-discover power monitoring devices.

### 1.4 Home Assistant MQTT Discovery

Home Assistant's MQTT Discovery protocol allows devices to self-announce their capabilities.
This is relevant because many homelab users run HA, and SubNetree can either consume HA's
discovery messages or publish its own.

**Discovery Topic Format:**

```text
<discovery_prefix>/<component>/<node_id>/<object_id>/config
```

Default `<discovery_prefix>` is `homeassistant`. For power sensors:

```text
homeassistant/sensor/<node_id>/<object_id>/config
```

**Discovery Payload for Power Sensor:**

```json
{
  "dev": {
    "ids": "smart_plug_001",
    "name": "Server Rack Plug",
    "mf": "Shelly",
    "mdl": "Plus Plug S"
  },
  "o": {
    "name": "subnetree",
    "sw": "0.3.0",
    "url": "https://github.com/HerbHall/subnetree"
  },
  "device_class": "power",
  "unit_of_measurement": "W",
  "value_template": "{{ value_json.apower }}",
  "unique_id": "smart_plug_001_power",
  "state_topic": "shellyplusplugs-001/status/switch:0"
}
```

**Relevant Device Classes:**

| Device Class | Unit | Description |
|-------------|------|-------------|
| `power` | W | Instantaneous power consumption |
| `energy` | kWh | Cumulative energy consumption |
| `voltage` | V | Supply voltage |
| `current` | A | Load current |
| `power_factor` | ratio | Power factor |
| `frequency` | Hz | Line frequency |

**Birth/Will Messages:** HA publishes `online`/`offline` to `homeassistant/status`. Devices
subscribe to this topic to know when to resend discovery payloads. SubNetree can use the same
mechanism to detect HA restarts and re-announce its entities.

### 1.5 Protocol Summary Table

| Protocol | Discovery | Payload Fields | Update Freq | Auth | Notes |
|----------|-----------|---------------|-------------|------|-------|
| Shelly Gen2 (MQTT) | mDNS + MQTT connect | `apower`, `voltage`, `current`, `pf`, `freq`, `aenergy.total` | On change / on demand | Broker user/pass | Native MQTT, no bridge needed |
| Tapo P110 (Zigbee2MQTT) | Z2M bridge/devices | `power`, `voltage`, `current`, `energy` | 10-60s configurable | Broker user/pass | Requires Zigbee coordinator |
| Zigbee2MQTT generic | Z2M bridge/devices | `power`, `voltage`, `current`, `energy` | 10-60s configurable | Broker user/pass | Consistent across all Z2M devices |
| HA MQTT Discovery | `homeassistant/+/+/config` | Varies by device_class | Varies by device | Broker user/pass | Standard for HA ecosystem |

## 2. SNMP PDU MIBs

### 2.1 APC (Schneider Electric) -- PowerNet-MIB

APC is the most common rack PDU vendor in homelabs. The PowerNet-MIB (enterprise OID
`.1.3.6.1.4.1.318`) covers Switched Rack PDUs, Metered Rack PDUs, and UPS units.

**MIB:** `PowerNet-MIB` (available from [APC download site](https://www.apc.com/us/en/download/document/SPD_ACZZ-0GHJMB_EN/))

**Key OIDs for Rack PDUs:**

| OID | Name | Description | Type |
|-----|------|-------------|------|
| `.1.3.6.1.4.1.318.1.1.12.1.16.0` | `rPDUIdentDevicePowerWatts` | Total PDU power draw (W) | Gauge |
| `.1.3.6.1.4.1.318.1.1.12.1.15.0` | `rPDUIdentDevicePowerVA` | Total PDU apparent power (VA) | Gauge |
| `.1.3.6.1.4.1.318.1.1.12.2.3.1.1.2.1` | `rPDULoadStatusLoad` | Phase/bank load (tenths of Amps) | Gauge |
| `.1.3.6.1.4.1.318.1.1.12.3.3.1.1.2.{n}` | `rPDUOutletStatusOutletState` | Per-outlet state (1=on, 2=off) | INTEGER |
| `.1.3.6.1.4.1.318.1.1.12.3.3.1.1.4.{n}` | `rPDUOutletStatusOutletName` | Outlet label | DisplayString |
| `.1.3.6.1.4.1.318.1.1.12.1.1.0` | `rPDUIdentName` | PDU name | DisplayString |
| `.1.3.6.1.4.1.318.1.1.12.1.2.0` | `rPDUIdentHardwareRev` | Hardware revision | DisplayString |
| `.1.3.6.1.4.1.318.1.1.12.1.3.0` | `rPDUIdentFirmwareRev` | Firmware revision | DisplayString |
| `.1.3.6.1.4.1.318.1.1.12.1.8.0` | `rPDUIdentDeviceNumOutlets` | Number of outlets | INTEGER |

**Per-Outlet Power (Metered-by-Outlet models only):**

| OID | Name | Description |
|-----|------|-------------|
| `.1.3.6.1.4.1.318.1.1.26.9.4.3.1.7.{n}` | `rPDU2OutletMeteredStatusPower` | Per-outlet power (W) |
| `.1.3.6.1.4.1.318.1.1.26.9.4.3.1.5.{n}` | `rPDU2OutletMeteredStatusCurrent` | Per-outlet current (0.1A) |
| `.1.3.6.1.4.1.318.1.1.26.9.4.3.1.6.{n}` | `rPDU2OutletMeteredStatusVoltage` | Per-outlet voltage (0.1V) |
| `.1.3.6.1.4.1.318.1.1.26.9.4.3.1.8.{n}` | `rPDU2OutletMeteredStatusEnergy` | Per-outlet energy (0.1 kWh) |

**SNMP Version Support:** v1, v2c (community string), v3 (USM auth). Most homelab PDUs use
v2c with a community string.

### 2.2 CyberPower

CyberPower PDUs use the `CPS-MIB` (enterprise OID `.1.3.6.1.4.1.3808`). Common in budget
homelab setups.

**MIB:** `CPS-MIB`

**Key OIDs:**

| OID | Name | Description | Type |
|-----|------|-------------|------|
| `.1.3.6.1.4.1.3808.1.1.3.3.5.0` | `ePDUStatusActivePower` | Total active power (W) | INTEGER |
| `.1.3.6.1.4.1.3808.1.1.3.3.1.0` | `ePDUStatusVoltage` | Input voltage (0.1V) | INTEGER |
| `.1.3.6.1.4.1.3808.1.1.3.3.2.0` | `ePDUStatusCurrent` | Total current (0.1A) | INTEGER |
| `.1.3.6.1.4.1.3808.1.1.3.3.6.0` | `ePDUStatusApparentPower` | Apparent power (VA) | INTEGER |
| `.1.3.6.1.4.1.3808.1.1.3.3.4.0` | `ePDUStatusFrequency` | Line frequency (0.1 Hz) | INTEGER |
| `.1.3.6.1.4.1.3808.1.1.3.3.3.0` | `ePDUStatusLoad` | Load percentage (%) | INTEGER |
| `.1.3.6.1.4.1.3808.1.1.3.5.3.1.1.2.{n}` | `ePDUOutletStatusOutletState` | Per-outlet state | INTEGER |
| `.1.3.6.1.4.1.3808.1.1.3.5.5.1.1.2.{n}` | `ePDUOutletMeteredStatusPower` | Per-outlet power (W) | INTEGER |

**Per-Outlet Support:** Available on Switched/Metered models (PDU41xxx series). Budget models
(PDU20xxx) only report total load.

**Authentication:** SNMPv2c with community string. SNMPv3 on newer firmware.

### 2.3 Tripp Lite

Tripp Lite uses the `TRIPPLITE-MIB` (enterprise OID `.1.3.6.1.4.1.850`). Common in
small business and homelab racks.

**MIB:** `TRIPPLITE-MIB`

**Key OIDs:**

| OID | Name | Description | Type |
|-----|------|-------------|------|
| `.1.3.6.1.4.1.850.1.1.3.1.3.3.1.1.2.{n}` | `tlpPduOutputActivePower` | Output active power per phase (W) | Unsigned32 |
| `.1.3.6.1.4.1.850.1.1.3.1.3.3.1.1.4.{n}` | `tlpPduOutputCurrent` | Output current per phase (0.01A) | Unsigned32 |
| `.1.3.6.1.4.1.850.1.1.3.1.3.3.1.1.3.{n}` | `tlpPduOutputApparentPower` | Output apparent power per phase (VA) | Unsigned32 |
| `.1.3.6.1.4.1.850.1.1.3.1.3.3.1.1.5.{n}` | `tlpPduOutputVoltage` | Output voltage per phase (0.1V) | Unsigned32 |
| `.1.3.6.1.4.1.850.1.1.3.1.3.2.1.1.6.{n}` | `tlpPduCircuitActivePower` | Per-circuit power (W) | Unsigned32 |
| `.1.3.6.1.4.1.850.1.1.3.1.3.1.0` | `tlpPduOutputPhaseCount` | Number of output phases | INTEGER |

**Per-Outlet Support:** Available on Monitored PDU models (PDUMNH series). Basic models
only report per-phase totals.

**SNMP Card:** Requires optional SNMPWEBCARD for SNMP access. Supports v1/v2c/v3.

### 2.4 Raritan (PX Series)

Raritan PX-series PDUs are enterprise-grade but found in prosumer homelabs, especially
second-hand from data center decommissions.

**MIB:** `PDU2-MIB` (enterprise OID `.1.3.6.1.4.1.13742`)

**Key OIDs:**

| OID | Name | Description | Type |
|-----|------|-------------|------|
| `.1.3.6.1.4.1.13742.6.5.2.3.1.4.{pdu}.{outlet}.1` | `outletActivePower` | Per-outlet active power (W) | Unsigned32 |
| `.1.3.6.1.4.1.13742.6.5.2.3.1.4.{pdu}.{outlet}.5` | `outletCurrent` | Per-outlet current (mA) | Unsigned32 |
| `.1.3.6.1.4.1.13742.6.5.2.3.1.4.{pdu}.{outlet}.4` | `outletVoltage` | Per-outlet voltage (mV) | Unsigned32 |
| `.1.3.6.1.4.1.13742.6.5.2.3.1.4.{pdu}.{outlet}.7` | `outletApparentPower` | Per-outlet apparent power (VA) | Unsigned32 |
| `.1.3.6.1.4.1.13742.6.5.2.3.1.4.{pdu}.{outlet}.40` | `outletActiveEnergy` | Per-outlet energy (Wh) | Unsigned32 |
| `.1.3.6.1.4.1.13742.6.5.4.3.1.4.{pdu}.{inlet}.1` | `inletActivePower` | Total inlet power (W) | Unsigned32 |
| `.1.3.6.1.4.1.13742.6.5.4.3.1.4.{pdu}.{inlet}.5` | `inletCurrent` | Total inlet current (mA) | Unsigned32 |

**Per-Outlet Support:** All PX models support per-outlet metering. This is the primary value
proposition of Raritan PDUs.

**Authentication:** SNMPv3 preferred. Supports user-based security model with AES/SHA.

### 2.5 SNMP PDU Summary Table

| Vendor | MIB | Enterprise OID | Per-Outlet Power | Per-Outlet Energy | Auth | Homelab Prevalence |
|--------|-----|---------------|-----------------|-------------------|------|-------------------|
| APC (Schneider) | PowerNet-MIB | `.1.3.6.1.4.1.318` | Yes (Metered-by-Outlet) | Yes (rPDU2) | v2c/v3 | Very High |
| CyberPower | CPS-MIB | `.1.3.6.1.4.1.3808` | Yes (Switched/Metered) | Limited | v2c/v3 | High |
| Tripp Lite | TRIPPLITE-MIB | `.1.3.6.1.4.1.850` | Per-circuit only | No | v1/v2c/v3 | Medium |
| Raritan | PDU2-MIB | `.1.3.6.1.4.1.13742` | Yes (all models) | Yes | v3 preferred | Low (prosumer) |

## 3. Electricity Rate Sources

### 3.1 US: EIA OpenData API

The U.S. Energy Information Administration provides a free, public API covering all U.S.
utility rates.

**Endpoint:** `https://api.eia.gov/v2/electricity/retail-sales/data/`

**Authentication:** Free API key required (register at [eia.gov/opendata](https://www.eia.gov/opendata/)).

**Example Query (Colorado residential rate):**

```text
GET https://api.eia.gov/v2/electricity/retail-sales/data/
    ?api_key=YOUR_KEY
    &frequency=monthly
    &data[]=price
    &facets[stateid][]=CO
    &facets[sectorid][]=RES
    &sort[0][column]=period
    &sort[0][direction]=desc
    &length=1
```

**Response (abbreviated):**

```json
{
  "response": {
    "total": 276,
    "data": [
      {
        "period": "2025-10",
        "stateid": "CO",
        "sectorid": "RES",
        "price": "15.62"
      }
    ]
  }
}
```

The `price` field is in **cents per kWh**. Divide by 100 for dollars.

**Available Facets:** `stateid` (US states + census regions), `sectorid` (RES, COM, IND, TRA, ALL).

**Frequency:** Monthly, quarterly, annual. Monthly data lags 2-3 months.

**Rate Limits:** Throttled per-key; excessive automated scraping triggers temporary suspension.

**Limitations:**
- State-level granularity only (not per-utility)
- 2-3 month publication lag
- Does not include time-of-use rate schedules
- Does not include demand charges or tiered pricing

### 3.2 EU: ENTSO-E Transparency Platform

The European Network of Transmission System Operators provides day-ahead electricity prices
for EU member states.

**Endpoint:** `https://web-api.tp.entsoe.eu/api` (RESTful XML API)

**Authentication:** Free registration required for a security token.

**Key Endpoint:** Day-ahead prices (A44 document type)

```text
GET https://web-api.tp.entsoe.eu/api
    ?securityToken=YOUR_TOKEN
    &documentType=A44
    &in_Domain=10YDE-RWENET---I
    &out_Domain=10YDE-RWENET---I
    &periodStart=202501010000
    &periodEnd=202501020000
```

**Response:** XML document with hourly price points in EUR/MWh. Divide by 1000 for EUR/kWh.

**Coverage:** All EU/EEA bidding zones. Prices are wholesale day-ahead; retail markup varies.

**Limitations:**
- XML-only responses (no JSON option)
- Wholesale prices only (does not include retail markup, taxes, network fees)
- Requires mapping bidding zones to geographic areas
- Rate limited to reasonable request volumes

### 3.3 UK: Octopus Energy API

Popular among UK homelab enthusiasts. Octopus provides a public REST API for tariff data.

**Endpoint:** `https://api.octopus.energy/v1/products/`

**Authentication:** API key (available to Octopus customers, but product listing is public).

**Key Endpoints:**

- List products: `GET /v1/products/`
- Product details: `GET /v1/products/{product_code}/`
- Tariff rates: `GET /v1/products/{product_code}/electricity-tariffs/{tariff_code}/standard-unit-rates/`

**Response (unit rates):**

```json
{
  "results": [
    {
      "value_exc_vat": 24.50,
      "value_inc_vat": 25.73,
      "valid_from": "2025-01-01T00:00:00Z",
      "valid_to": "2025-03-31T23:59:59Z"
    }
  ]
}
```

Rates are in **pence per kWh**. Divide by 100 for GBP.

**Agile Tariff:** Octopus Agile provides half-hourly pricing, making it the most granular
option for time-of-use cost calculation. Very popular with homelab energy trackers.

**Limitations:**
- UK-only
- Agile rates available only for Octopus Agile customers
- Standard tariff listing is public but varies by region (GSP group)

### 3.4 Manual Configuration (Recommended Default)

For the initial implementation, manual rate entry is the most practical approach:

**Configuration Model:**

```yaml
power:
  default_rate:
    currency: "USD"
    rate_per_kwh: 0.1562
  time_of_use:
    enabled: false
    schedules:
      - name: "Peak"
        rate_per_kwh: 0.25
        hours: "14:00-20:00"
        days: "mon-fri"
      - name: "Off-Peak"
        rate_per_kwh: 0.10
        hours: "20:00-14:00"
        days: "mon-fri"
      - name: "Weekend"
        rate_per_kwh: 0.08
        days: "sat-sun"
  billing_cycle_day: 1
```

**Rationale:**
- Works for all countries and currencies immediately
- No external API dependency
- Users know their own rate (printed on utility bills)
- Time-of-use support covers most utility rate structures
- API-based rate lookup can be added as an enhancement in later phases

### 3.5 Rate Source Recommendation

| Priority | Approach | Effort | Coverage |
|----------|----------|--------|----------|
| **Phase 1** | Manual rate entry ($/kWh) | Low | Global |
| **Phase 2** | EIA API integration (US auto-lookup) | Medium | US states |
| **Phase 3** | Octopus API (UK) + ENTSO-E (EU) | Medium | UK + EU |
| **Future** | Community-contributed rate plugins | Low (SDK) | Extensible |

## 4. Proposed Plugin Architecture

### 4.1 Plugin Identity

```go
func (p *Power) Info() plugin.PluginInfo {
    return plugin.PluginInfo{
        Name:         "power",
        Version:      "0.1.0",
        Description:  "Power monitoring and energy cost tracking",
        Dependencies: []string{"recon"},    // for device matching
        Required:     false,
        Roles:        []string{"power_monitoring", "energy_tracking"},
        APIVersion:   plugin.APIVersionCurrent,
    }
}
```

**Implemented Interfaces:**

- `plugin.Plugin` -- lifecycle management
- `plugin.HTTPProvider` -- REST API endpoints
- `plugin.EventSubscriber` -- reacts to device discovery events
- `plugin.HealthChecker` -- reports MQTT connection status, SNMP poll health
- `plugin.Validator` -- validates MQTT broker config, rate configuration

**Dependencies:**

- `recon` (required) -- device discovery for MAC/IP-to-device matching
- `pulse` (optional) -- correlation between power anomalies and health status
- External: MQTT broker connection (user-configured)

### 4.2 Data Model

**Migration 1: Core power tables**

```sql
CREATE TABLE IF NOT EXISTS power_sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL,       -- 'mqtt_shelly', 'mqtt_zigbee2mqtt', 'mqtt_generic', 'snmp'
    config TEXT NOT NULL,            -- JSON: broker, topic, OID, credentials ref
    device_id TEXT,                  -- linked device (nullable until matched)
    enabled INTEGER NOT NULL DEFAULT 1,
    last_seen_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_power_sources_device ON power_sources(device_id);

CREATE TABLE IF NOT EXISTS power_readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id TEXT NOT NULL REFERENCES power_sources(id) ON DELETE CASCADE,
    device_id TEXT,                  -- denormalized for query performance
    watts REAL NOT NULL,
    voltage REAL,
    current_amps REAL,
    power_factor REAL,
    energy_kwh REAL,                -- cumulative counter from device
    source_type TEXT NOT NULL,      -- 'mqtt' or 'snmp'
    recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_power_readings_source_time
    ON power_readings(source_id, recorded_at);
CREATE INDEX IF NOT EXISTS idx_power_readings_device_time
    ON power_readings(device_id, recorded_at);

CREATE TABLE IF NOT EXISTS power_config (
    id TEXT PRIMARY KEY,
    device_id TEXT UNIQUE,          -- NULL = global default
    rate_per_kwh REAL NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    billing_cycle_day INTEGER NOT NULL DEFAULT 1,
    tou_enabled INTEGER NOT NULL DEFAULT 0,
    tou_schedules TEXT,             -- JSON array of time-of-use schedules
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_power_config_device ON power_config(device_id);

CREATE TABLE IF NOT EXISTS power_budgets (
    id TEXT PRIMARY KEY,
    device_id TEXT,                  -- NULL = global budget
    monthly_budget_kwh REAL NOT NULL,
    alert_threshold_pct REAL NOT NULL DEFAULT 80.0,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_power_budgets_device ON power_budgets(device_id);
```

**Data Retention:** `power_readings` will accumulate quickly (one row per reading per source).
A maintenance routine should:

- Aggregate readings older than 7 days into hourly averages
- Aggregate readings older than 30 days into daily averages
- Delete raw readings older than 90 days (configurable)

This mirrors the Pulse module's maintenance pattern.

### 4.3 Event Topics

**Published by Power module:**

```go
const (
    TopicReadingReceived = "power.reading.received"
    TopicBudgetExceeded  = "power.budget.exceeded"
    TopicAnomalyDetected = "power.anomaly.detected"
    TopicSourceOnline    = "power.source.online"
    TopicSourceOffline   = "power.source.offline"
)
```

**Consumed by Power module:**

```go
const (
    TopicDeviceDiscovered = "recon.device.discovered"  // auto-match sources to devices
    TopicDeviceRemoved    = "recon.device.removed"     // unlink power source
)
```

**Event Payloads:**

```go
// PowerReadingEvent is published on TopicReadingReceived
type PowerReadingEvent struct {
    SourceID   string    `json:"source_id"`
    DeviceID   string    `json:"device_id,omitempty"`
    Watts      float64   `json:"watts"`
    Voltage    float64   `json:"voltage,omitempty"`
    CurrentA   float64   `json:"current_amps,omitempty"`
    EnergyKWh  float64   `json:"energy_kwh,omitempty"`
    RecordedAt time.Time `json:"recorded_at"`
}

// BudgetExceededEvent is published on TopicBudgetExceeded
type BudgetExceededEvent struct {
    DeviceID       string  `json:"device_id"`
    BudgetKWh      float64 `json:"budget_kwh"`
    CurrentKWh     float64 `json:"current_kwh"`
    ThresholdPct   float64 `json:"threshold_pct"`
    ProjectedKWh   float64 `json:"projected_kwh"`
}

// AnomalyEvent is published on TopicAnomalyDetected
type AnomalyEvent struct {
    DeviceID    string  `json:"device_id"`
    CurrentW    float64 `json:"current_watts"`
    BaselineW   float64 `json:"baseline_watts"`
    DeviationPct float64 `json:"deviation_pct"`
    Message     string  `json:"message"`
}
```

### 4.4 API Endpoints

All endpoints mounted under `/api/v1/power/`:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/power/readings/{device_id}` | Time-series power data for a device |
| `GET` | `/power/summary` | Fleet-wide power summary |
| `GET` | `/power/devices` | Per-device power status list |
| `GET` | `/power/sources` | List configured power sources (MQTT/SNMP) |
| `POST` | `/power/sources` | Add a new power source |
| `PUT` | `/power/sources/{id}` | Update a power source |
| `DELETE` | `/power/sources/{id}` | Remove a power source |
| `GET` | `/power/config/{device_id}` | Get rate/budget config for a device |
| `PUT` | `/power/config/{device_id}` | Set rate, currency, billing cycle |
| `GET` | `/power/config` | Get global default config |
| `PUT` | `/power/config` | Set global default config |
| `GET` | `/power/budgets` | List all budgets |
| `POST` | `/power/budgets` | Create a budget |
| `PUT` | `/power/budgets/{id}` | Update a budget |
| `DELETE` | `/power/budgets/{id}` | Delete a budget |
| `GET` | `/power/cost/estimate` | Monthly cost projection |
| `GET` | `/power/cost/history` | Historical cost by period |

**Query Parameters for `/power/readings/{device_id}`:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `since` | ISO8601 | 24h ago | Start of time range |
| `until` | ISO8601 | now | End of time range |
| `resolution` | string | `auto` | `raw`, `minute`, `hour`, `day` |

**Response Example (`GET /power/summary`):**

```json
{
  "total_watts": 847.3,
  "total_devices": 12,
  "estimated_monthly_kwh": 622.8,
  "estimated_monthly_cost": 97.28,
  "currency": "USD",
  "rate_per_kwh": 0.1562,
  "top_consumers": [
    {"device_id": "abc123", "device_name": "NAS", "watts": 285.4},
    {"device_id": "def456", "device_name": "Proxmox Node 1", "watts": 210.1},
    {"device_id": "ghi789", "device_name": "Network Switch", "watts": 95.0}
  ],
  "trend": "stable",
  "measured_at": "2026-02-14T10:30:00Z"
}
```

### 4.5 MQTT Integration Design

```text
+------------------+       +------------------+       +------------------+
|  MQTT Broker     |       |  Power Plugin    |       |  SubNetree DB    |
|  (Mosquitto)     |<----->|  MQTT Subscriber |------>|  power_readings  |
+------------------+       +------------------+       +------------------+
                                   |
                                   | parse payload
                                   | match to device
                                   v
                           +------------------+
                           |  Event Bus       |
                           |  power.reading.  |
                           |  received        |
                           +------------------+
```

**Subscription Strategy:**

1. Subscribe to configured topics (user provides MQTT broker + topic patterns)
2. Parse incoming JSON payloads using source-type-specific parsers:
   - `mqtt_shelly`: Extract `apower`, `voltage`, `current`, `aenergy.total`
   - `mqtt_zigbee2mqtt`: Extract `power`, `voltage`, `current`, `energy`
   - `mqtt_generic`: User-configurable JSON path mappings
3. Match source to device by MAC address, IP address, or user-configured mapping
4. Store reading in `power_readings` table
5. Publish `power.reading.received` event

**Connection Management:**

- Maintain persistent MQTT connection with auto-reconnect
- Support multiple brokers (some users have separate brokers for IoT vs. infra)
- TLS support for broker connections
- Health check reports MQTT connection state

### 4.6 SNMP Integration Design

```text
+------------------+       +------------------+       +------------------+
|  Rack PDU        |       |  Power Plugin    |       |  SubNetree DB    |
|  (APC/Cyber/etc) |<----->|  SNMP Poller     |------>|  power_readings  |
+------------------+       +------------------+       +------------------+
                                   |
                                   | walk OID tables
                                   | per-outlet mapping
                                   v
                           +------------------+
                           |  Event Bus       |
                           |  power.reading.  |
                           |  received        |
                           +------------------+
```

**Polling Strategy:**

1. Poll configured PDUs on schedule (default: 30 seconds)
2. Use vendor-specific OID mappings (auto-detect by sysObjectID)
3. Walk outlet tables for per-outlet readings
4. Map outlet numbers to devices (user-configured mapping: outlet 5 = NAS)
5. Store aggregated and per-outlet readings
6. Publish events per reading

**Auto-Detection:** On first poll, read `sysObjectID.0` to identify vendor, then select
appropriate MIB/OID mapping automatically.

## 5. Dashboard UX Design

### 5.1 Power Overview Widget (Dashboard Home)

**Placement:** Main dashboard, alongside existing device count and health widgets.

**Content:**

- Large number: total rack/home power draw in watts (e.g., "847 W")
- Secondary line: estimated monthly cost (e.g., "$97.28/mo")
- Trend indicator: arrow up/down/stable compared to 24h average
- Mini sparkline: last 24h power draw (optional, deferred to Phase 3)
- Color coding: green (under budget), yellow (approaching 80% budget), red (over budget)

**Interaction:** Click navigates to `/power` detail page.

**Data Source:** `GET /power/summary`

### 5.2 Power Detail Page (`/power`)

**Layout:** Full-width page accessible from sidebar navigation.

**Section 1: Fleet Summary Bar**

- Total power draw (W), estimated monthly cost, total energy this billing cycle (kWh)
- Budget progress bar (if global budget configured)
- Rate display (current $/kWh, editable inline)

**Section 2: Per-Device Power Table**

Sortable, filterable table with columns:

| Column | Description | Sortable |
|--------|-------------|----------|
| Device Name | Linked device name (or source name if unmatched) | Yes |
| Current (W) | Latest power reading | Yes |
| Daily (kWh) | Energy consumed today | Yes |
| Monthly (kWh) | Energy consumed this billing cycle | Yes |
| Monthly Cost | Estimated cost this billing cycle | Yes |
| Trend | Sparkline (7-day) or arrow indicator | No |
| Status | Online/Offline/Stale | Yes |

**Section 3: Fleet Power Chart**

- Stacked area chart showing power draw over time (selectable range: 24h, 7d, 30d)
- Each device is a colored band in the stack
- Hover shows device-specific values at that time point
- Total line overlay

**Interaction:**

- Click a device row to navigate to that device's detail page
- Filter by: all devices, MQTT sources only, SNMP sources only
- Search/filter by device name

### 5.3 Device Power Section (Device Detail Page)

**Placement:** New tab or collapsible section on the existing device detail page, shown only
for devices with a linked power source.

**Content:**

- **Power History Chart:** Line chart showing watts over time (24h, 7d, 30d selectable)
- **Energy Accumulation Chart:** Bar chart showing daily kWh consumption
- **Cost Accumulation:** Running total for current billing cycle with daily breakdown
- **Budget Progress Bar:** If a device-specific budget is set, show used/remaining with
  percentage and projected end-of-cycle usage
- **Power Source Info:** Source type (MQTT/SNMP), last reading time, connection status
- **Statistics Summary:**
  - Average power (24h, 7d, 30d)
  - Peak power (24h, 7d, 30d)
  - Idle power (minimum sustained reading)
  - Total energy this month
  - Estimated monthly cost at current rate

### 5.4 Alerts and Notifications

**Budget Threshold Alerts:**

- Trigger when energy consumption reaches configured percentage of monthly budget
- Default thresholds: 80%, 90%, 100%
- Alert includes: device name, current usage, budget, projected end-of-month usage
- Published as `power.budget.exceeded` event (integrates with Pulse notification channels)

**Anomaly Detection Alerts:**

- Trigger when a device's power draw exceeds 2x its 7-day rolling average for more than
  5 minutes (configurable multiplier and duration)
- Use case: NAS started consuming 400W instead of usual 120W (possible disk failure,
  runaway process)
- Alert includes: device name, current watts, baseline watts, deviation percentage
- Published as `power.anomaly.detected` event

**Source Offline Alerts:**

- Trigger when an MQTT source stops publishing for longer than 3x its expected interval
- Trigger when an SNMP poll fails consecutively (3 failures = alert)
- Published as `power.source.offline` event

## 6. Implementation Phases

### Phase 1: MQTT Smart Plug Monitoring (MVP)

**Scope:** Core plugin skeleton + MQTT ingestion + manual rate config + basic dashboard.

**Tasks:**

1. Plugin skeleton implementing `Plugin`, `HTTPProvider`, `EventSubscriber`, `HealthChecker`
2. Database migrations (all tables from Section 4.2)
3. MQTT client with support for Shelly Gen2 and Zigbee2MQTT payload parsing
4. `power_readings` store with time-range queries
5. Manual rate configuration API (`PUT /power/config`)
6. Summary endpoint (`GET /power/summary`)
7. Per-device readings endpoint (`GET /power/readings/{device_id}`)
8. Power Overview Widget on dashboard
9. Basic Power Detail Page with per-device table
10. Contract tests and unit tests

**Estimated Complexity:** Medium. Primary risk is MQTT client reliability and reconnection
handling.

### Phase 2: SNMP PDU + Cost Tracking

**Scope:** SNMP polling + per-outlet device mapping + historical cost calculation + budgets.

**Tasks:**

1. SNMP poller with APC and CyberPower MIB support
2. Auto-detection of PDU vendor via `sysObjectID`
3. Per-outlet to device mapping (UI configuration)
4. Budget management API and threshold alerting
5. Cost history endpoint (`GET /power/cost/history`)
6. Data retention/aggregation maintenance routine
7. Device Power Section on device detail page
8. Fleet Power Chart (stacked area chart)
9. Anomaly detection (simple threshold-based)

**Estimated Complexity:** Medium-High. SNMP vendor differences add testing surface.

### Phase 3: Advanced Features

**Scope:** API-based rate lookup + time-of-use scheduling + HA discovery + Insight integration.

**Tasks:**

1. EIA API integration for US rate auto-lookup
2. Time-of-use rate schedule support
3. Home Assistant MQTT Discovery (publish SubNetree power entities to HA)
4. Power cost correlation in Insight module (AI/analytics)
5. Export power data (CSV, Prometheus metrics endpoint)
6. Mini sparklines in dashboard widget and device table
7. Tripp Lite and Raritan SNMP MIB support
8. Multi-broker MQTT support

**Estimated Complexity:** Medium. API integrations are straightforward; HA discovery requires
careful adherence to the protocol specification.

## 7. Risk Assessment

### Dependencies

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| MQTT broker not available | Medium | High (no data) | Clear setup docs; test with embedded broker option |
| SNMP PDU firmware variations | High | Medium | Start with APC only; community-contributed MIB mappings |
| Rate API downtime/deprecation | Low | Low | Manual rate always works as fallback |
| SQLite write contention (high-frequency readings) | Medium | Medium | Batch inserts; WAL mode; 10s minimum read interval |

### Complexity

| Area | Complexity | Rationale |
|------|-----------|-----------|
| MQTT payload parsing | Low | Well-documented JSON formats; small number of variants |
| SNMP OID walking | Medium | Vendor-specific OID trees; table indexing varies |
| Device matching | Medium | MAC/IP correlation requires Recon integration |
| Data retention | Low | Same pattern as Pulse maintenance routine |
| Time-of-use rate calc | Medium | Schedule intersection with timestamp ranges |
| Anomaly detection | Low-Medium | Simple rolling average; no ML needed |

### Unknowns

- **MQTT client library selection for Go:** Options include `eclipse/paho.mqtt.golang` (v1,
  stable, widely used) and `mochi-mqtt/server` (if embedding a broker). Paho v1 is the
  safest choice.
- **SNMP library:** `gosnmp/gosnmp` is the standard Go SNMP library. Well-maintained,
  supports v1/v2c/v3.
- **Write throughput:** With 20 power sources reporting every 10 seconds, that is 120
  writes/minute. SQLite in WAL mode handles this comfortably. At 100+ sources, batch
  inserts become important.
- **Shelly Gen2 MQTT message rate:** Shelly devices publish status on every change. Under
  fluctuating loads, this could be multiple messages per second. The plugin should
  debounce or sample at a configurable minimum interval (default: 10 seconds).

## 8. References

### Vendor Documentation

- [Shelly Gen2 Switch Component API](https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/Switch/)
- [Zigbee2MQTT MQTT Topics and Messages](https://www.zigbee2mqtt.io/guide/usage/mqtt_topics_and_messages.html)
- [Zigbee2MQTT Supported Devices](https://www.zigbee2mqtt.io/supported-devices/)
- [Home Assistant MQTT Integration](https://www.home-assistant.io/integrations/mqtt/)
- [Home Assistant MQTT Discovery](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery)

### SNMP MIBs

- [APC PowerNet-MIB (Observium Mirror)](https://mibs.observium.org/mib/PowerNet-MIB/)
- [APC MIB Downloads (Schneider Electric)](https://www.apc.com/us/en/download/document/SPD_ACZZ-0GHJMB_EN/)
- [CyberPower MIB Documentation](https://www.cyberpowersystems.com/products/software/snmp/)
- [Tripp Lite SNMP Documentation](https://www.tripplite.com/support/snmp)
- [Raritan PX PDU2-MIB](https://www.raritan.com/support/product/px3)

### Electricity Rate APIs

- [EIA OpenData API v2 Documentation](https://www.eia.gov/opendata/documentation.php)
- [EIA Electricity Retail Sales Data](https://api.eia.gov/v2/electricity/retail-sales/)
- [ENTSO-E Transparency Platform API](https://transparency.entsoe.eu/content/static_content/Static%20content/web%20api/Guide.html)
- [Octopus Energy Developer API](https://developer.octopus.energy/)

### Go Libraries

- [eclipse/paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang) -- MQTT v3.1.1 client
- [gosnmp/gosnmp](https://github.com/gosnmp/gosnmp) -- SNMP v1/v2c/v3 library
