/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { getStatus } from '@/lib/api'

export type SiteModelCallsConfig = {
  enabled: boolean
  models: string[]
}

export const DEFAULT_SITE_MODEL_CALLS_CONFIG: SiteModelCallsConfig = {
  enabled: false,
  models: [],
}

export function normalizeSiteModelCallsConfig(
  config: SiteModelCallsConfig
): SiteModelCallsConfig {
  return {
    enabled: Boolean(config.enabled),
    models: [
      ...new Set(
        (Array.isArray(config.models) ? config.models : [])
          .map((model) => model.trim())
          .filter(Boolean)
      ),
    ].sort((a, b) => a.localeCompare(b)),
  }
}

export function parseSiteModelCallsConfig(raw: unknown): SiteModelCallsConfig {
  if (!raw) return DEFAULT_SITE_MODEL_CALLS_CONFIG
  try {
    const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw
    if (!parsed || typeof parsed !== 'object') {
      return DEFAULT_SITE_MODEL_CALLS_CONFIG
    }
    const value = parsed as Partial<SiteModelCallsConfig>
    return normalizeSiteModelCallsConfig({
      enabled: value.enabled === true,
      models: Array.isArray(value.models) ? value.models : [],
    })
  } catch {
    return DEFAULT_SITE_MODEL_CALLS_CONFIG
  }
}

export function serializeSiteModelCallsConfig(
  config: SiteModelCallsConfig
): string {
  return JSON.stringify(normalizeSiteModelCallsConfig(config))
}

export async function getFreshSiteModelCallsConfig() {
  const status = await getStatus()
  return parseSiteModelCallsConfig(status?.AllSiteModelCalls)
}
