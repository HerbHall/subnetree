/**
 * Default token values for dark and light base modes.
 * Extracted from design-tokens.css. Used by the theme editor to show
 * current defaults and detect which tokens have been overridden.
 *
 * Note: Values that reference var(--nv-*) in CSS are resolved here to
 * their final computed values for use in color pickers.
 */

import type { ThemeTokens } from '@/api/themes'

export const DARK_DEFAULTS: Record<string, string> = {
  // Backgrounds
  '--nv-bg-root': '#0c1a0e',
  '--nv-bg-surface': '#0f1a10',
  '--nv-bg-card': '#1a2e1c',
  '--nv-bg-elevated': '#1f3320',
  '--nv-bg-hover': 'rgba(74, 222, 128, 0.06)',
  '--nv-bg-active': 'rgba(74, 222, 128, 0.10)',
  '--nv-bg-selected': 'rgba(74, 222, 128, 0.08)',

  // Text
  '--nv-text-primary': '#f5f0e8',
  '--nv-text-secondary': '#9ca389',
  '--nv-text-muted': '#5c6650',
  '--nv-text-accent': '#4ade80',
  '--nv-text-warm': '#c4a77d',
  '--nv-text-inverse': '#0c1a0e',

  // Borders
  '--nv-border-subtle': 'rgba(74, 222, 128, 0.08)',
  '--nv-border-default': 'rgba(74, 222, 128, 0.12)',
  '--nv-border-strong': 'rgba(74, 222, 128, 0.20)',
  '--nv-border-focus': '#4ade80',

  // Buttons
  '--nv-btn-primary-bg': '#16a34a',
  '--nv-btn-primary-hover': '#22c55e',
  '--nv-btn-primary-text': '#ffffff',
  '--nv-btn-danger-bg': '#991b1b',
  '--nv-btn-danger-hover': '#b91c1c',
  '--nv-btn-danger-text': '#fecaca',

  // Inputs
  '--nv-input-bg': '#0f1a10',
  '--nv-input-border': 'rgba(74, 222, 128, 0.12)',
  '--nv-input-focus': '#4ade80',
  '--nv-input-text': '#f5f0e8',
  '--nv-input-placeholder': '#5c6650',

  // Sidebar
  '--nv-sidebar-bg': '#0c1a0e',
  '--nv-sidebar-item': '#9ca389',
  '--nv-sidebar-active': '#4ade80',
  '--nv-sidebar-active-bg': 'rgba(74, 222, 128, 0.08)',

  // Status
  '--nv-status-online': '#4ade80',
  '--nv-status-degraded': '#c4a77d',
  '--nv-status-offline': '#ef4444',
  '--nv-status-unknown': '#9ca389',

  // Charts
  '--nv-chart-green': '#4ade80',
  '--nv-chart-amber': '#c4a77d',
  '--nv-chart-sage': '#9ca389',
  '--nv-chart-red': '#ef4444',
  '--nv-chart-blue': '#60a5fa',
  '--nv-chart-grid': 'rgba(74, 222, 128, 0.06)',

  // Typography
  '--nv-font-sans': "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'Inter', Helvetica, Arial, sans-serif",
  '--nv-font-mono': "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace",

  // Spacing
  '--nv-radius-sm': '4px',
  '--nv-radius-md': '8px',
  '--nv-radius-lg': '12px',
  '--nv-radius-xl': '16px',

  // Shadows
  '--nv-shadow-sm': '0 1px 2px rgba(0, 0, 0, 0.3)',
  '--nv-shadow-md': '0 4px 6px rgba(0, 0, 0, 0.25)',
  '--nv-shadow-lg': '0 10px 25px rgba(0, 0, 0, 0.3)',
  '--nv-shadow-glow': '0 0 20px rgba(74, 222, 128, 0.15)',

  // Transitions
  '--nv-transition-fast': '150ms ease',
  '--nv-transition-normal': '250ms ease',
  '--nv-transition-slow': '350ms ease',
}

export const LIGHT_DEFAULTS: Record<string, string> = {
  // Backgrounds
  '--nv-bg-root': '#f5f5f0',
  '--nv-bg-surface': '#eeede6',
  '--nv-bg-card': '#ffffff',
  '--nv-bg-elevated': '#ffffff',
  '--nv-bg-hover': 'rgba(22, 101, 52, 0.04)',
  '--nv-bg-active': 'rgba(22, 101, 52, 0.08)',
  '--nv-bg-selected': 'rgba(22, 101, 52, 0.06)',

  // Text
  '--nv-text-primary': '#1a2e1c',
  '--nv-text-secondary': '#6b7a56',
  '--nv-text-muted': '#9ca389',
  '--nv-text-accent': '#16a34a',
  '--nv-text-warm': '#92734e',
  '--nv-text-inverse': '#f5f0e8',

  // Borders
  '--nv-border-subtle': 'rgba(22, 101, 52, 0.06)',
  '--nv-border-default': 'rgba(22, 101, 52, 0.12)',
  '--nv-border-strong': 'rgba(22, 101, 52, 0.20)',
  '--nv-border-focus': '#16a34a',

  // Buttons
  '--nv-btn-primary-bg': '#16a34a',
  '--nv-btn-primary-hover': '#15803d',
  '--nv-btn-primary-text': '#ffffff',
  '--nv-btn-danger-bg': '#991b1b',
  '--nv-btn-danger-hover': '#b91c1c',
  '--nv-btn-danger-text': '#fecaca',

  // Inputs
  '--nv-input-bg': '#ffffff',
  '--nv-input-border': 'rgba(22, 101, 52, 0.12)',
  '--nv-input-focus': '#16a34a',
  '--nv-input-text': '#1a2e1c',
  '--nv-input-placeholder': '#9ca389',

  // Sidebar
  '--nv-sidebar-bg': '#eeede6',
  '--nv-sidebar-item': '#6b7a56',
  '--nv-sidebar-active': '#15803d',
  '--nv-sidebar-active-bg': 'rgba(22, 101, 52, 0.08)',

  // Status (same in both modes)
  '--nv-status-online': '#4ade80',
  '--nv-status-degraded': '#c4a77d',
  '--nv-status-offline': '#ef4444',
  '--nv-status-unknown': '#9ca389',

  // Charts (same in both modes)
  '--nv-chart-green': '#4ade80',
  '--nv-chart-amber': '#c4a77d',
  '--nv-chart-sage': '#9ca389',
  '--nv-chart-red': '#ef4444',
  '--nv-chart-blue': '#60a5fa',
  '--nv-chart-grid': 'rgba(22, 101, 52, 0.06)',

  // Typography (same in both modes)
  '--nv-font-sans': "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'Inter', Helvetica, Arial, sans-serif",
  '--nv-font-mono': "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace",

  // Spacing (same in both modes)
  '--nv-radius-sm': '4px',
  '--nv-radius-md': '8px',
  '--nv-radius-lg': '12px',
  '--nv-radius-xl': '16px',

  // Shadows
  '--nv-shadow-sm': '0 1px 2px rgba(0, 0, 0, 0.05)',
  '--nv-shadow-md': '0 4px 6px rgba(0, 0, 0, 0.07)',
  '--nv-shadow-lg': '0 10px 25px rgba(0, 0, 0, 0.1)',
  '--nv-shadow-glow': '0 0 20px rgba(22, 101, 52, 0.1)',

  // Transitions (same in both modes)
  '--nv-transition-fast': '150ms ease',
  '--nv-transition-normal': '250ms ease',
  '--nv-transition-slow': '350ms ease',
}

/**
 * Get the default value for a CSS variable given a base mode.
 */
export function getDefault(varName: string, baseMode: 'dark' | 'light'): string {
  const defaults = baseMode === 'dark' ? DARK_DEFAULTS : LIGHT_DEFAULTS
  return defaults[varName] ?? ''
}

/**
 * Get all defaults for a base mode.
 */
export function getDefaults(baseMode: 'dark' | 'light'): Record<string, string> {
  return baseMode === 'dark' ? DARK_DEFAULTS : LIGHT_DEFAULTS
}

/**
 * Token category definitions for the theme editor UI.
 * Maps category names to the CSS variable names they contain.
 */
export const TOKEN_CATEGORIES: Record<string, { label: string; vars: string[] }> = {
  backgrounds: {
    label: 'Backgrounds',
    vars: [
      '--nv-bg-root', '--nv-bg-surface', '--nv-bg-card', '--nv-bg-elevated',
      '--nv-bg-hover', '--nv-bg-active', '--nv-bg-selected',
    ],
  },
  text: {
    label: 'Text',
    vars: [
      '--nv-text-primary', '--nv-text-secondary', '--nv-text-muted',
      '--nv-text-accent', '--nv-text-warm', '--nv-text-inverse',
    ],
  },
  borders: {
    label: 'Borders',
    vars: [
      '--nv-border-subtle', '--nv-border-default', '--nv-border-strong', '--nv-border-focus',
    ],
  },
  buttons: {
    label: 'Buttons',
    vars: [
      '--nv-btn-primary-bg', '--nv-btn-primary-hover', '--nv-btn-primary-text',
      '--nv-btn-danger-bg', '--nv-btn-danger-hover', '--nv-btn-danger-text',
    ],
  },
  inputs: {
    label: 'Inputs',
    vars: [
      '--nv-input-bg', '--nv-input-border', '--nv-input-focus',
      '--nv-input-text', '--nv-input-placeholder',
    ],
  },
  sidebar: {
    label: 'Sidebar',
    vars: [
      '--nv-sidebar-bg', '--nv-sidebar-item', '--nv-sidebar-active', '--nv-sidebar-active-bg',
    ],
  },
  status: {
    label: 'Status',
    vars: [
      '--nv-status-online', '--nv-status-degraded', '--nv-status-offline', '--nv-status-unknown',
    ],
  },
  charts: {
    label: 'Charts',
    vars: [
      '--nv-chart-green', '--nv-chart-amber', '--nv-chart-sage',
      '--nv-chart-red', '--nv-chart-blue', '--nv-chart-grid',
    ],
  },
  typography: {
    label: 'Typography',
    vars: ['--nv-font-sans', '--nv-font-mono'],
  },
  spacing: {
    label: 'Spacing & Radius',
    vars: ['--nv-radius-sm', '--nv-radius-md', '--nv-radius-lg', '--nv-radius-xl'],
  },
  effects: {
    label: 'Effects',
    vars: [
      '--nv-shadow-sm', '--nv-shadow-md', '--nv-shadow-lg', '--nv-shadow-glow',
      '--nv-transition-fast', '--nv-transition-normal', '--nv-transition-slow',
    ],
  },
}

/**
 * Maps each theme layer to the token categories it contains.
 * Used for layer badges and grouping in the editor.
 */
export const LAYER_CATEGORIES: Record<string, { label: string; categories: string[] }> = {
  colors: {
    label: 'Colors',
    categories: ['backgrounds', 'text', 'borders', 'buttons', 'inputs', 'sidebar', 'status', 'charts'],
  },
  typography: {
    label: 'Typography',
    categories: ['typography'],
  },
  shape: {
    label: 'Shape',
    categories: ['spacing'],
  },
  effects: {
    label: 'Effects',
    categories: ['effects'],
  },
}

/**
 * Flatten a ThemeTokens object into a flat CSS variable map.
 * { backgrounds: { 'bg-root': '#fff' } } -> { '--nv-bg-root': '#fff' }
 */
export function flattenTokens(tokens: ThemeTokens): Record<string, string> {
  const result: Record<string, string> = {}
  const entries = Object.entries(tokens) as [string, Record<string, string> | undefined][]
  for (const [, categoryTokens] of entries) {
    if (!categoryTokens) continue
    for (const [key, value] of Object.entries(categoryTokens)) {
      result[`--nv-${key}`] = value
    }
  }
  return result
}

/**
 * Unflatten a CSS variable map into ThemeTokens categories.
 * { '--nv-bg-root': '#fff' } -> { backgrounds: { 'bg-root': '#fff' } }
 */
export function unflattenTokens(flat: Record<string, string>): Record<string, Record<string, string>> {
  const result: Record<string, Record<string, string>> = {}
  for (const [varName, value] of Object.entries(flat)) {
    const key = varName.replace('--nv-', '')
    for (const [category, def] of Object.entries(TOKEN_CATEGORIES)) {
      if (def.vars.includes(varName)) {
        if (!result[category]) result[category] = {}
        result[category][key] = value
        break
      }
    }
  }
  return result
}
