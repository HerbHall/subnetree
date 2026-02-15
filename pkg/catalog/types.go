// Package catalog provides a curated catalog of complementary homelab tools
// with hardware-tier filtering for SubNetree's recommendation engine.
package catalog

// HardwareTier represents a target deployment environment.
type HardwareTier int

const (
	TierSBC     HardwareTier = 0 // Pi, SBC (1 GB)
	TierMiniPC  HardwareTier = 1 // Intel N100 (8-32 GB)
	TierNAS     HardwareTier = 2 // Synology/QNAP (Scout only)
	TierCluster HardwareTier = 3 // Proxmox cluster
	TierSMB     HardwareTier = 4 // Small business server
)

// Category groups tools by their primary function.
type Category string

const (
	CategoryMonitoring          Category = "monitoring"
	CategoryInfrastructure      Category = "infrastructure"
	CategoryDNS                 Category = "dns"
	CategoryHomeAutomation      Category = "home-automation"
	CategoryNotifications       Category = "notifications"
	CategoryDashboard           Category = "dashboard"
	CategoryCMDB                Category = "cmdb"
	CategoryDocumentation       Category = "documentation"
	CategoryNetwork             Category = "network"
	CategoryBackup              Category = "backup"
	CategorySecurity            Category = "security"
	CategoryMedia               Category = "media"
	CategoryVPN                 Category = "vpn"
	CategoryAutomation          Category = "automation"
	CategoryContainerManagement Category = "container-management"
)

// IntegrationStatus describes how a tool integrates with SubNetree.
type IntegrationStatus string

const (
	IntegrationShipped  IntegrationStatus = "shipped"  // Already works
	IntegrationPlanned  IntegrationStatus = "planned"  // On roadmap
	IntegrationPossible IntegrationStatus = "possible" // Feasible, not scheduled
)

// CatalogEntry represents a single tool in the recommendation catalog.
type CatalogEntry struct {
	Name              string            `yaml:"name" json:"name"`
	Description       string            `yaml:"description" json:"description"`
	Category          Category          `yaml:"category" json:"category"`
	GitHubURL         string            `yaml:"github_url" json:"github_url"`
	Website           string            `yaml:"website" json:"website"`
	DockerImage       string            `yaml:"docker_image" json:"docker_image"`
	Stars             int               `yaml:"stars" json:"stars"`
	License           string            `yaml:"license" json:"license"`
	Language          string            `yaml:"language" json:"language"`
	MinRAMMB          int               `yaml:"min_ram_mb" json:"min_ram_mb"`
	SupportedTiers    []HardwareTier    `yaml:"supported_tiers" json:"supported_tiers"`
	Protocols         []string          `yaml:"protocols" json:"protocols"`
	IntegrationStatus IntegrationStatus `yaml:"integration_status" json:"integration_status"`
	IntegrationNotes  string            `yaml:"integration_notes" json:"integration_notes"`
	Tags              []string          `yaml:"tags" json:"tags"`
}
