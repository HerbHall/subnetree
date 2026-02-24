# 10. Hardware inventory schema for multi-source device profiling

Date: 2026-02-21

## Status

Accepted

## Context

SubNetree collects hardware information from multiple sources: Scout agents
(real-time profiling), manual entry (user overrides), and potentially future
SNMP/WMI collectors. Hardware data is heterogeneous -- a device has one CPU
info but multiple storage devices and GPUs. Existing schema needs to track
which source provided each piece of data and preserve manual overrides when
automated collection runs.

The previous approach of storing hardware as a JSON blob in the device table
limited querying (SQLite JSON functions are limited in pure Go) and made
aggregation across a fleet impractical.

## Decision

Use structured relational tables instead of JSON blobs for hardware inventory:

1. **Core hardware table**: `recon_device_hardware` stores CPU, memory, and OS
   information. One row per device.

2. **Storage devices table**: `recon_device_storage` stores disk/volume information.
   One row per disk (multiple rows per device).

3. **GPU devices table**: `recon_device_gpu` stores GPU information. One row per GPU
   (multiple rows per device).

4. **Collection provenance**: All three tables include:
   - `collection_source` field ("scout", "manual", "snmp", "wmi") indicating the
     data source
   - `collected_at` timestamp tracking when the data was collected

5. **Manual override protection**: When `collection_source = "manual"`, automated
   collection never overwrites. The `UpsertDeviceHardware` store method checks
   the existing source before updating. Manual data persists across automated scans.

6. **Fleet aggregation**: A hardware summary endpoint aggregates across the fleet:
   - Total device count and OS distribution
   - CPU/memory statistics (min, max, average across fleet)
   - Storage utilization by device type
   - GPU inventory and VRAM totals

7. **Device hardware API**: `GET /api/v1/recon/devices/{id}/hardware` returns a
   combined profile including CPU, memory, storage list, and GPU list. The response
   structure is tailored for UI and API consumption, not the raw database schema.

## Consequences

### Positive

- **SQL-queryable hardware data** -- standard SQL can answer questions like
  "find all devices with >16GB RAM" or "group devices by OS" without JSON parsing
- **Multi-source tracking** -- audit trail showing which source provided each
  piece of data
- **Manual override preservation** -- user edits are protected from automated
  overwrite, maintaining trust in the data
- **Flexible cardinality** -- each table can independently store 1-to-many
  relationships (one device → multiple disks, multiple GPUs)
- **MCP tool support** -- `query_devices` MCP tool can filter by hardware specs
  directly via SQL

### Negative

- **Migration complexity** -- converting from JSON to three tables requires a
  careful schema migration with data transformation
- **Adding hardware categories** -- future hardware types (network adapters,
  TPMs, RAID controllers) require new migrations and store methods
- **More JOIN queries** -- retrieving a complete device profile now requires
  JOINs across three tables instead of a single JSON fetch

### Neutral

- Follows the existing pattern of `recon_` prefixed tables (consistent with
  `recon_devices`, `recon_device_services`, etc.)
- Hardware data is exposed through both REST API (`/api/v1/recon/devices/{id}/hardware`)
  and MCP tools (`get_hardware_profile`, `query_devices`)
- Scout agent collects hardware profiles during check-in and stores them via
  the Dispatch module's `UpsertDeviceHardware` call

## Alternatives Considered

### Alternative 1: JSON columns in device table

Store hardware as a JSON blob directly in the `recon_devices` table:

```sql
ALTER TABLE recon_devices ADD COLUMN hardware JSON;
-- hardware: { "cpu": {...}, "memory": {...}, "storage": [...], "gpus": [...] }
```

**Rejected** because:
- SQLite's JSON functions are limited (`json_extract` for simple paths, but no
  powerful aggregation)
- Pure Go SQLite bindings (modernc.org/sqlite) don't expose extended JSON operators
- Fleet aggregation queries (group by OS, aggregate memory) become impossible
  in pure SQL
- Each update requires parsing, modifying, and re-serializing the JSON blob

### Alternative 2: Single hardware table with JSON sub-fields

One `device_hardware` table with JSON columns for storage and GPUs:

```sql
CREATE TABLE device_hardware (
  id INTEGER PRIMARY KEY,
  device_id INTEGER,
  cpu_info JSON,
  memory_mb INTEGER,
  os_type TEXT,
  storage_devices JSON,  -- array of disk objects
  gpus JSON              -- array of GPU objects
);
```

**Rejected** because it combines the worst of both approaches:
- Relational overhead (foreign key, indexed lookups)
- JSON query limitations (can't efficiently search within the arrays)
- Still requires application-level parsing for multi-GPU or multi-disk iteration

### Alternative 3: Generic key-value attribute store

Store hardware as flexible key-value pairs:

```sql
CREATE TABLE device_attributes (
  device_id INTEGER,
  attribute_name TEXT,      -- "cpu.cores", "memory.total_mb", "os.type"
  attribute_value TEXT,
  collection_source TEXT,
  collected_at TIMESTAMP
);
```

**Rejected** because:
- Loses type safety (everything is a string, requires casting)
- Aggregation queries become verbose (nested CASE statements)
- No foreign key constraints or schema definition
- Difficult to validate (no schema enforcement)

## Related Decisions

- **ADR-0002 SQLite-First Database** -- Hardware inventory confirms the
  SQLite-first approach by using relational tables (not JSON) for queryable data
- **Hardware Profiles Feature** -- Implemented in Sprint 4 (PR #441-443) with
  this schema design
