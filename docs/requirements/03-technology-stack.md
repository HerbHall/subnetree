## Technology Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Server | Go (1.25+) | Performance, single binary deployment, strong networking stdlib |
| Agent | Go | Same language as server, cross-compiles to all targets |
| Dashboard | React + TypeScript (Vite) | Largest ecosystem, rich component libraries |
| UI Components | shadcn/ui + Tailwind CSS | Customizable, not a dependency, modern styling |
| UI State | TanStack Query + Zustand | TanStack for server state, Zustand for client state |
| UI Charts | Recharts | Composable React charting library |
| Agent Communication | gRPC + Protobuf (buf) | Efficient binary protocol, bidirectional streaming, code generation |
| Real-time UI | WebSocket | Push updates to dashboard without polling |
| Configuration | Viper (YAML) | Standard Go config library, env var support |
| Logging | Zap | High-performance structured logging |
| Database (Phase 1) | SQLite via modernc.org/sqlite | Pure Go (no CGo), zero-config, cross-compilation friendly |
| Database (Phase 2+) | PostgreSQL + TimescaleDB | Time-series metrics at scale, multi-tenant support |
| HTTP Routing | net/http (stdlib) | No unnecessary dependencies for Phase 1 |
| Authentication | Local (bcrypt) + JWT | Local auth default, OIDC optional |
| Remote Desktop | Apache Guacamole (Docker) | Apache 2.0 licensed, proven RDP/VNC gateway |
| SSH Terminal | xterm.js + Go SSH library | Browser-based SSH terminal |
| HTTP Proxy | Go reverse proxy (stdlib) | Access device web interfaces through server |
| SNMP | gosnmp | Pure Go SNMP library |
| MQTT | Eclipse Paho Go | MQTT client for IoT device communication |
| Metrics Exposition | Prometheus client_golang | Industry standard metrics format |
| Tailscale API | tailscale-client-go-v2 | MIT licensed Tailscale API client for tailnet device discovery |
| Graph Operations | dominikbraun/graph | Apache 2.0 licensed generic graph library for dependency resolution, topology computation, cycle detection |
| Statistical Analysis | gonum.org/v1/gonum | BSD-3 licensed numerical computing: statistics, linear algebra, FFT for analytics engine |
| LLM Client (OpenAI) | sashabaranov/go-openai | Apache 2.0 licensed OpenAI API client with streaming, function calling, embeddings (Phase 3) |
| LLM Client (Anthropic) | liushuangls/go-anthropic | Apache 2.0 licensed Anthropic Claude API client with tool use, streaming (Phase 3) |
| Model Inference | yalue/onnxruntime_go | MIT licensed ONNX Runtime bindings for Go, supports x64 + ARM64 (Phase 4) |
| Proto Management | buf | Modern protobuf toolchain, linting, breaking change detection |
