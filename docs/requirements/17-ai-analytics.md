## AI & Analytics Strategy

### Design Philosophy

AI in NetVantage follows the same principles as the rest of the platform: **optional, progressive, and practical**. Every AI feature is a plugin that can be disabled entirely. The core monitoring system works perfectly without any AI capabilities. AI enhances the experience -- it never gates it.

Research into how "AI" works in production monitoring tools (Datadog, Dynatrace, Auvik, Moogsoft) reveals that the most valuable features are fundamentally **statistical algorithms and graph traversal**, not deep learning. Dynamic baselining, anomaly detection, and topology-aware alert correlation deliver 80%+ of the value at a fraction of the computational cost of neural networks.

### Three-Tier Architecture

AI capabilities are organized into three tiers that scale with available hardware. Each tier is independently optional.

| Tier | Name | Compute | Additional RAM | Phase | Default State |
|------|------|---------|---------------|-------|---------------|
| 1 | Statistical Analytics | Pure Go (gonum), in-process | 20–200 MB | 2 | Enabled on `medium`+ profiles |
| 2 | On-Device Inference | ONNX Runtime, in-process | 100–500 MB | 4 | Disabled (opt-in) |
| 3 | LLM Integration | External API or local Ollama | Negligible (API) or 4–16 GB (local) | 3 | Disabled (opt-in, BYOK) |

### Tier 1: Statistical Analytics Engine (Phase 2)

The built-in **Insight** plugin provides always-on statistical analysis using pure Go and gonum. No external dependencies, no GPU, no API keys. This is the highest-value tier.

**Adaptive Baselines:**
- EWMA (Exponentially Weighted Moving Average) for all monitored metrics -- learns each device's "normal" automatically
- Time-of-day and day-of-week seasonal baselines (Holt-Winters family) -- understands that Monday 9 AM traffic differs from Sunday 3 AM
- Per-device, per-metric baselines that adapt to gradual drift (growing traffic over months) while detecting sudden anomalies
- Learning period: 7 days minimum before baselines are considered stable; flags metrics as "learning" in the dashboard

**Anomaly Detection:**
- Z-score detection with configurable sensitivity (default: 3σ) -- flags metrics that deviate significantly from their baseline
- Multivariate correlation: CPU spike + memory increase + network drop occurring simultaneously on one device is a single anomaly, not three
- Change-point detection (CUSUM algorithm) -- identifies when a metric's behavior permanently shifts ("the network got slower starting Tuesday")

**Trend Detection & Capacity Forecasting:**
- Linear regression on sliding windows -- "this disk will be full in 3 days at current growth rate"
- Capacity forecast with confidence intervals displayed on device detail pages
- Proactive warnings before resources reach critical thresholds (not just when they cross a line)

**Topology-Aware Alert Correlation:**
- When a switch goes down, suppress alerts for all devices behind that switch
- Group correlated alerts into a single incident with identified root cause
- Learn device dependency relationships from alert timing patterns (if Device A always alerts 2 seconds before Device B, they are likely dependent)
- Reduce alert volume by 50–90% during cascading failures

**Alert Pattern Learning:**
- Track which alerts are acknowledged vs investigated vs result in action
- Reduce sensitivity for metrics that consistently produce false positives
- Flag "flapping" devices (rapid up/down cycling) for special handling
- Maintenance window prediction: "this device has been rebooted at 2 AM every Sunday for the last 4 weeks"

**Resource requirements (Tier 1):**

| Deployment Size | Additional RAM | Additional CPU | Notes |
|----------------|---------------|----------------|-------|
| 50 devices | 20–50 MB | < 1% of 1 core | Runs on any hardware |
| 200 devices | 80–200 MB | 2–5% of 1 core | Comfortable on RPi 5 |
| 500 devices | 200–500 MB | 5–10% of 1 core | Needs 2 GB+ free RAM |

**Implementation:** gonum for statistical functions + hand-rolled algorithms (~50–300 lines each for EWMA, CUSUM, Holt-Winters, Z-score). No external library dependencies beyond gonum.

### Tier 2: On-Device Model Inference (Phase 4)

Optional machine learning models that run locally via ONNX Runtime. Models are trained offline (in Python) and shipped as ONNX files. Go loads and runs inference without Python.

**Device Fingerprinting:**
- Input features: MAC vendor (OUI), ICMP response characteristics (TTL, timing), open ports, SNMP sysDescr, hostname patterns, mDNS service types
- Output: device type classification with confidence score (e.g., "Raspberry Pi running Home Assistant, 94% confidence")
- Model: Random Forest or Gradient Boosted Trees, 5–20 MB ONNX file
- Inference time: < 10 ms per device on CPU
- Training data: crowd-sourced from opt-in telemetry (anonymized) or user-corrected classifications

**Traffic Classification:**
- Input: flow features (bytes/packets per flow, duration, port, protocol)
- Output: traffic category (web browsing, video streaming, VoIP, backup, IoT telemetry)
- Model: Gradient Boosted Trees (XGBoost/LightGBM), 10–30 MB ONNX file
- Inference time: < 5 ms per flow batch on CPU

**Prerequisites:**
- Minimum 8 GB RAM (model + ONNX Runtime overhead ~100–200 MB on top of base server)
- x64 or ARM64 architecture (ONNX Runtime supports both via onnxruntime_go)
- 500 MB disk for model weight files
- `large` performance profile or explicit opt-in

**Why Phase 4:** Training data for device fingerprinting and traffic classification will not exist at meaningful scale until the platform is deployed to hundreds of users. Tier 1 statistical methods handle 80% of the use cases without training data.

### Tier 3: LLM Integration (Phase 3)

Optional integration with large language models for natural language interaction. Follows a "bring your own API key" model -- NetVantage never requires a paid AI subscription.

**Natural Language Querying:**
- "Show me all devices that had CPU above 90% last Tuesday"
- "Which switches had the most packet loss this month?"
- "What changed on the network in the last 24 hours?"
- Implementation: LLM translates natural language to a structured API query via function calling / tool use, executes it, returns formatted results
- Latency: 1–5 seconds for API round-trips (acceptable for interactive use)

**Incident Summarization:**
- Feed alert timeline + metric data to LLM, receive human-readable incident summary
- Example: "At 3:42 AM, switch SW-CORE-01 began experiencing elevated packet loss (12%) on interface Gi0/1. This correlated with a 40% bandwidth spike from subnet 10.1.5.0/24. The issue resolved at 4:15 AM when the traffic subsided."
- Displayed on alert detail pages and in email/webhook notifications

**Report Generation:**
- Weekly/monthly network health reports in natural language
- Aggregated metrics summarized with highlights and recommendations
- Scheduled (not interactive), so latency is irrelevant

**Configuration Assistance:**
- "Help me configure SNMP v3 for this Cisco switch"
- Context-aware suggestions based on device inventory and existing configuration

**Supported Providers:**

| Provider | Library | Auth | Notes |
|----------|---------|------|-------|
| OpenAI (GPT-4o, GPT-4o-mini) | sashabaranov/go-openai | API key | Lowest latency, function calling support |
| Anthropic (Claude Sonnet, Haiku) | liushuangls/go-anthropic | API key | Tool use support, strong reasoning |
| Ollama (local) | net/http (REST API) | None | Fully offline, requires 8–16 GB RAM for 7B model |

**Privacy Controls:**
- **Data anonymization:** Before sending data to external APIs, replace real IPs with opaque device IDs, strip hostnames, aggregate metrics. Configurable level: none / partial / full.
- **Local-only mode:** Restrict to Ollama only -- no data leaves the premises.
- **Audit logging:** Every LLM API call is logged with the query (sanitized) and the response.
- **No training data contribution:** API providers (OpenAI, Anthropic) do not train on API data per their data processing agreements.

**Cost estimates (external API):**

| Usage Level | Queries/Day | Monthly Cost (GPT-4o-mini) | Monthly Cost (Claude Haiku) |
|-------------|------------|---------------------------|----------------------------|
| Light | 10 | $3–15 | $5–20 |
| Medium | 50 + daily reports | $15–75 | $25–100 |
| Heavy | Interactive + reports | $50–200 | $75–300 |

### Analytics by Performance Profile

| Profile | Tier 1 (Statistical) | Tier 2 (Inference) | Tier 3 (LLM API) | Tier 3 (Local Ollama) |
|---------|---------------------|-------------------|-------------------|----------------------|
| **micro** | Disabled | Disabled | Disabled | Disabled |
| **small** | Basic (EWMA, Z-score only) | Disabled | Available (if API key set) | Disabled |
| **medium** | Full Tier 1 (all algorithms) | Disabled | Available | Disabled |
| **large** | Full Tier 1 | Available (opt-in) | Available | Available (7B model, background tasks) |

### AI Plugin Architecture

The Insight plugin (Tier 1) and optional Tier 2/3 plugins follow the standard NetVantage plugin pattern:

- Implements `Plugin` interface (core lifecycle)
- Implements `EventSubscriber` to receive metric/alert/device events via `PublishAsync` (never blocks metric collection)
- Implements `AnalyticsProvider` for structured analysis queries from other plugins and the API
- Implements `HTTPProvider` to expose `/api/v1/analytics/*` REST endpoints
- Implements `HealthChecker` to report model load status, inference latency, learning progress
- Implements `Validator` to check model files, RAM availability, API key validity at startup
- Declares `Prerequisites` for RAM, disk, and optional external dependencies (ONNX Runtime, Ollama)

**Event subscriptions:**

| Event | AI Use |
|-------|--------|
| `pulse.metrics.collected` | Anomaly detection, baseline updates, trend analysis |
| `recon.device.discovered` | Device classification, risk scoring |
| `recon.device.updated` | Reclassification with new metadata |
| `pulse.alert.triggered` | Alert correlation, pattern learning |
| `pulse.alert.resolved` | Recovery time distribution learning |
| `vault.credential.accessed` | Anomalous access pattern detection |

**New API endpoints (exposed by Insight plugin):**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/analytics/anomalies` | List detected anomalies with severity, device, metric |
| `GET` | `/api/v1/analytics/anomalies/{device_id}` | Anomalies for a specific device |
| `GET` | `/api/v1/analytics/forecasts/{device_id}` | Capacity forecasts for device metrics |
| `GET` | `/api/v1/analytics/correlations` | Active alert correlation groups |
| `GET` | `/api/v1/analytics/baselines/{device_id}` | Current learned baselines for a device |
| `POST` | `/api/v1/analytics/query` | Natural language query (Tier 3, if configured) |

### Foundation Requirements (Phase 1)

To enable AI features in later phases without retrofit, Phase 1 must establish:

1. **Metrics format:** Every collected metric includes `timestamp`, `device_id`, `metric_name`, `value`, and optional `tags` map. This uniform format is what the analytics engine consumes.
2. **Event bus async support:** `PublishAsync` handlers for slow consumers (already planned). Analytics plugins are the primary consumer of this capability.
3. **Analytics role interface:** Define `AnalyticsProvider` in `pkg/roles/analytics.go` (interface only, no implementation in Phase 1). This establishes the contract that Tier 1/2/3 plugins implement.
4. **Baseline storage schema:** Reserve a `analytics_baselines` table prefix in the database migration plan. The Insight plugin will create its tables during Phase 2 initialization.

These are interface definitions and data format conventions -- zero implementation overhead in Phase 1.
