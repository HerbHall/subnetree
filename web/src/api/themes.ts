import { api } from './client'

export type ThemeLayer = 'colors' | 'typography' | 'shape' | 'effects'

/**
 * CSS token overrides organized by category.
 * Only overridden tokens are stored -- missing keys use base mode defaults.
 */
export interface ThemeTokens {
  backgrounds?: Record<string, string>
  text?: Record<string, string>
  borders?: Record<string, string>
  buttons?: Record<string, string>
  inputs?: Record<string, string>
  sidebar?: Record<string, string>
  status?: Record<string, string>
  charts?: Record<string, string>
  typography?: Record<string, string>
  spacing?: Record<string, string>
  effects?: Record<string, string>
}

/**
 * Complete theme definition as returned by the API.
 */
export interface ThemeDefinition {
  id: string
  name: string
  description: string
  base_mode: 'dark' | 'light'
  version: number
  created_at: string
  updated_at: string
  built_in: boolean
  layers?: ThemeLayer[]
  tokens: ThemeTokens
}

/**
 * Request body for creating a new theme.
 */
export interface CreateThemeRequest {
  name: string
  description?: string
  base_mode: 'dark' | 'light'
  layers?: ThemeLayer[]
  tokens: ThemeTokens
}

/**
 * Request body for updating an existing theme.
 */
export interface UpdateThemeRequest {
  name?: string
  description?: string
  tokens?: ThemeTokens
}

export async function listThemes(): Promise<ThemeDefinition[]> {
  return api.get<ThemeDefinition[]>('/settings/themes')
}

export async function getTheme(id: string): Promise<ThemeDefinition> {
  return api.get<ThemeDefinition>(`/settings/themes/${id}`)
}

export async function createTheme(theme: CreateThemeRequest): Promise<ThemeDefinition> {
  return api.post<ThemeDefinition>('/settings/themes', theme)
}

export async function updateTheme(id: string, theme: UpdateThemeRequest): Promise<ThemeDefinition> {
  return api.put<ThemeDefinition>(`/settings/themes/${id}`, theme)
}

export async function deleteTheme(id: string): Promise<void> {
  await api.delete(`/settings/themes/${id}`)
}

export async function getActiveTheme(): Promise<{ theme_id: string }> {
  return api.get<{ theme_id: string }>('/settings/themes/active')
}

export async function setActiveTheme(themeId: string): Promise<void> {
  await api.put('/settings/themes/active', { theme_id: themeId })
}
