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
import { api } from '@/lib/api'

import type {
  ProhibitedWordConfigResponse,
  ProhibitedWordSummaryResponse,
} from './types'

export async function getProhibitedWordConfig() {
  const response = await api.get<ProhibitedWordConfigResponse>(
    '/api/prohibited-word-detection/config'
  )
  return response.data
}

export async function updateProhibitedWordConfig(keywords: string[]) {
  const response = await api.put<ProhibitedWordConfigResponse>(
    '/api/prohibited-word-detection/config',
    { keywords }
  )
  return response.data
}

export async function getProhibitedWordSummary(page: number, pageSize: number) {
  const response = await api.get<ProhibitedWordSummaryResponse>(
    '/api/prohibited-word-detection/summary',
    { params: { p: page, page_size: pageSize } }
  )
  return response.data
}

export async function clearProhibitedWordStats() {
  const response = await api.delete('/api/prohibited-word-detection/stats')
  return response.data as { success: boolean; message?: string }
}
