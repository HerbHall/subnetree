import { type LucideIcon } from 'lucide-react'
import {
  Globe,
  Lock,
  Terminal,
  Database,
  Mail,
  FileText,
  Monitor,
  Radio,
  HardDrive,
  Play,
  Home,
  Shield,
  Container,
  Gauge,
  Key,
  Cog,
} from 'lucide-react'

/** Category for grouping and prioritizing service icons. */
export type ServiceCategory =
  | 'web'
  | 'remote'
  | 'database'
  | 'mail'
  | 'file'
  | 'media'
  | 'iot'
  | 'monitoring'
  | 'other'

/**
 * Describes a well-known network service with its display metadata.
 * Named ServiceIconInfo to avoid conflict with api/types.ts ServiceInfo.
 */
export interface ServiceIconInfo {
  name: string
  icon: LucideIcon
  category: ServiceCategory
}

/**
 * Maps well-known ports to service info.
 * Covers the most common homelab services.
 */
export const SERVICE_PORT_MAP: Record<number, ServiceIconInfo> = {
  // Web
  80: { name: 'HTTP', icon: Globe, category: 'web' },
  443: { name: 'HTTPS', icon: Lock, category: 'web' },
  8080: { name: 'HTTP Alt', icon: Globe, category: 'web' },
  8443: { name: 'HTTPS Alt', icon: Lock, category: 'web' },

  // Remote access
  22: { name: 'SSH', icon: Terminal, category: 'remote' },
  3389: { name: 'RDP', icon: Monitor, category: 'remote' },
  5900: { name: 'VNC', icon: Monitor, category: 'remote' },

  // Database
  3306: { name: 'MySQL', icon: Database, category: 'database' },
  5432: { name: 'PostgreSQL', icon: Database, category: 'database' },
  6379: { name: 'Redis', icon: Database, category: 'database' },
  27017: { name: 'MongoDB', icon: Database, category: 'database' },

  // Mail
  25: { name: 'SMTP', icon: Mail, category: 'mail' },
  587: { name: 'SMTP/TLS', icon: Mail, category: 'mail' },
  993: { name: 'IMAPS', icon: Mail, category: 'mail' },

  // File sharing
  445: { name: 'SMB', icon: FileText, category: 'file' },
  21: { name: 'FTP', icon: FileText, category: 'file' },
  2049: { name: 'NFS', icon: HardDrive, category: 'file' },

  // Media
  8096: { name: 'Jellyfin', icon: Play, category: 'media' },
  32400: { name: 'Plex', icon: Play, category: 'media' },
  8989: { name: 'Sonarr', icon: Play, category: 'media' },
  7878: { name: 'Radarr', icon: Play, category: 'media' },

  // IoT / Home Automation
  8123: { name: 'Home Assistant', icon: Home, category: 'iot' },
  1883: { name: 'MQTT', icon: Radio, category: 'iot' },
  8883: { name: 'MQTT/TLS', icon: Radio, category: 'iot' },

  // Monitoring
  9090: { name: 'Prometheus', icon: Gauge, category: 'monitoring' },
  3000: { name: 'Grafana', icon: Gauge, category: 'monitoring' },
  9100: { name: 'Node Exporter', icon: Gauge, category: 'monitoring' },

  // Security / Auth
  8200: { name: 'Vault', icon: Key, category: 'other' },

  // Containers
  2375: { name: 'Docker API', icon: Container, category: 'other' },
  2376: { name: 'Docker TLS', icon: Container, category: 'other' },
  9000: { name: 'Portainer', icon: Container, category: 'other' },

  // DNS
  53: { name: 'DNS', icon: Globe, category: 'other' },
  853: { name: 'DNS/TLS', icon: Globe, category: 'other' },

  // Proxies / Reverse Proxy
  81: { name: 'Nginx PM', icon: Shield, category: 'web' },
  8081: { name: 'Traefik', icon: Shield, category: 'web' },

  // Misc
  161: { name: 'SNMP', icon: Cog, category: 'monitoring' },
}

/** Category priority for badge ordering (lower = more important). */
const CATEGORY_PRIORITY: Record<ServiceCategory, number> = {
  web: 1,
  remote: 2,
  database: 3,
  monitoring: 4,
  iot: 5,
  media: 6,
  mail: 7,
  file: 8,
  other: 9,
}

/**
 * Look up service info by port number.
 * Returns undefined for unknown ports.
 */
export function getServiceByPort(port: number): ServiceIconInfo | undefined {
  return SERVICE_PORT_MAP[port]
}

/**
 * Get service badges for a list of open ports.
 * Returns up to maxBadges services, prioritized by category importance.
 * Deduplicates icons so the same visual isn't repeated.
 */
export function getServiceBadges(
  ports: number[],
  maxBadges: number = 3,
): ServiceIconInfo[] {
  const seen = new Set<string>()

  return ports
    .map((p) => getServiceByPort(p))
    .filter((s): s is ServiceIconInfo => s !== undefined)
    .sort(
      (a, b) =>
        (CATEGORY_PRIORITY[a.category] ?? 99) -
        (CATEGORY_PRIORITY[b.category] ?? 99),
    )
    .filter((s) => {
      // Deduplicate by name so we don't show e.g. two Globe icons
      if (seen.has(s.name)) return false
      seen.add(s.name)
      return true
    })
    .slice(0, maxBadges)
}
