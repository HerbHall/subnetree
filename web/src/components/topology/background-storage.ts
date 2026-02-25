export type BackgroundType = 'none' | 'grid' | 'dots' | 'image'

export interface TopologyBackgroundSettings {
  type: BackgroundType
  opacity: number
  imageData?: string
}

export const DEFAULT_BACKGROUND_SETTINGS: TopologyBackgroundSettings = {
  type: 'none',
  opacity: 30,
}

const STORAGE_KEY = 'subnetree-topology-background'

/** Load background settings from localStorage. */
export function loadBackgroundSettings(): TopologyBackgroundSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return DEFAULT_BACKGROUND_SETTINGS
    const parsed = JSON.parse(raw) as TopologyBackgroundSettings
    // Validate type field
    if (!['none', 'grid', 'dots', 'image'].includes(parsed.type)) {
      return DEFAULT_BACKGROUND_SETTINGS
    }
    return {
      type: parsed.type,
      opacity: Math.max(10, Math.min(100, parsed.opacity ?? 30)),
      imageData: parsed.imageData,
    }
  } catch {
    return DEFAULT_BACKGROUND_SETTINGS
  }
}

/** Save background settings to localStorage. */
export function saveBackgroundSettings(settings: TopologyBackgroundSettings): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
  } catch {
    // localStorage full or unavailable -- silently ignore
  }
}
