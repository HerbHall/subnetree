## Event System

### Event Bus

Inter-plugin communication via typed publish/subscribe. Synchronous by default (handlers run in publisher's goroutine) with `PublishAsync` available for slow handlers.

```go
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    PublishAsync(ctx context.Context, event Event)
    Subscribe(topic string, handler EventHandler) (unsubscribe func())
    SubscribeAll(handler EventHandler) (unsubscribe func())
}

type Event struct {
    Topic     string    // "{plugin}.{entity}.{action}" e.g., "recon.device.discovered"
    Source    string    // Plugin name that emitted the event
    Timestamp time.Time
    Payload   any       // Type depends on topic (documented per constant)
}
```

### Core Event Topics

| Topic | Payload Type | Emitter | Subscribers |
|-------|-------------|---------|-------------|
| `recon.device.discovered` | `*models.Device` | Recon | Pulse, Gateway, Topology |
| `recon.device.updated` | `*models.Device` | Recon | Pulse, Dashboard |
| `recon.device.lost` | `DeviceLostEvent` | Recon | Pulse, Dashboard |
| `recon.scan.started` | `*models.ScanResult` | Recon | Dashboard |
| `recon.scan.completed` | `*models.ScanResult` | Recon | Dashboard |
| `pulse.alert.triggered` | `Alert` | Pulse | Notifiers, Dashboard |
| `pulse.alert.resolved` | `Alert` | Pulse | Notifiers, Dashboard |
| `pulse.metrics.collected` | `MetricsBatch` | Pulse | Data Exporters, Analytics |
| `dispatch.agent.connected` | `*models.AgentInfo` | Dispatch | Dashboard |
| `dispatch.agent.disconnected` | `*models.AgentInfo` | Dispatch | Dashboard |
| `dispatch.agent.enrolled` | `*models.AgentInfo` | Dispatch | Recon, Dashboard |
| `vault.credential.created` | `CredentialEvent` | Vault | Audit Log |
| `vault.credential.accessed` | `CredentialEvent` | Vault | Audit Log |
| `system.plugin.unhealthy` | `PluginHealthEvent` | Registry | Dashboard, Notifiers |
