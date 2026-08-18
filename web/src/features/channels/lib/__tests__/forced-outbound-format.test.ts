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
import { describe, expect, test } from 'vitest'

import { CHANNEL_TYPE_NEW_API } from '../../constants'
import type { Channel } from '../../types'
import {
  buildSettingJSON,
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
  FORCED_OUTBOUND_FORMAT_AUTO,
  supportsForcedOutboundFormat,
  transformChannelToFormDefaults,
  type ChannelFormValues,
} from '../channel-form'
import { hasAdvancedSettingsErrors } from '../channel-form-errors'

function createValidForm(
  overrides: Partial<ChannelFormValues> = {}
): ChannelFormValues {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'Protocol channel',
    key: 'test-key',
    models: 'gpt-5',
    ...overrides,
  }
}

function createChannel(
  type: number,
  setting: Record<string, unknown>
): Channel {
  return {
    id: 1,
    type,
    key: '',
    openai_organization: null,
    test_model: null,
    status: 1,
    name: 'Protocol channel',
    weight: 0,
    created_time: 0,
    test_time: 0,
    response_time: 0,
    base_url: 'https://upstream.example',
    other: '',
    balance: 0,
    balance_updated_time: 0,
    models: 'gpt-5',
    group: 'default',
    used_quota: 0,
    model_mapping: null,
    status_code_mapping: null,
    priority: 0,
    auto_ban: 1,
    other_info: '',
    tag: null,
    setting: JSON.stringify(setting),
    param_override: null,
    header_override: null,
    remark: '',
    max_input_tokens: 0,
    channel_info: {
      is_multi_key: false,
      multi_key_size: 0,
      multi_key_polling_index: 0,
      multi_key_mode: 'random',
    },
    settings: '{}',
  }
}

describe('forced outbound format form settings', () => {
  test('uses auto by default and omits it from setting JSON', () => {
    const formData = createValidForm({
      pass_through_body_enabled: true,
    })

    expect(formData.forced_outbound_format).toBe(FORCED_OUTBOUND_FORMAT_AUTO)
    expect(JSON.parse(buildSettingJSON(formData))).toMatchObject({
      pass_through_body_enabled: true,
    })
    expect(JSON.parse(buildSettingJSON(formData))).not.toHaveProperty(
      'forced_outbound_format'
    )
  })

  test('serializes a forced format and disables request body passthrough', () => {
    const setting = JSON.parse(
      buildSettingJSON(
        createValidForm({
          forced_outbound_format: 'openai_responses',
          pass_through_body_enabled: true,
        })
      )
    )

    expect(setting).toMatchObject({
      forced_outbound_format: 'openai_responses',
      pass_through_body_enabled: false,
    })
  })

  test('loads supported settings and presents passthrough as disabled', () => {
    const defaults = transformChannelToFormDefaults(
      createChannel(CHANNEL_TYPE_NEW_API, {
        forced_outbound_format: 'claude',
        pass_through_body_enabled: true,
      })
    )

    expect(defaults.forced_outbound_format).toBe('claude')
    expect(defaults.pass_through_body_enabled).toBe(false)
  })

  test('ignores invalid or unsupported stored values', () => {
    expect(
      transformChannelToFormDefaults(
        createChannel(14, { forced_outbound_format: 'gemini' })
      ).forced_outbound_format
    ).toBe(FORCED_OUTBOUND_FORMAT_AUTO)
    expect(
      transformChannelToFormDefaults(
        createChannel(1, { forced_outbound_format: 'unknown' })
      ).forced_outbound_format
    ).toBe(FORCED_OUTBOUND_FORMAT_AUTO)
  })

  test('accepts only OpenAI and New API channel types', () => {
    expect(supportsForcedOutboundFormat(1)).toBe(true)
    expect(supportsForcedOutboundFormat(CHANNEL_TYPE_NEW_API)).toBe(true)
    expect(supportsForcedOutboundFormat(14)).toBe(false)

    expect(
      channelFormSchema.safeParse(
        createValidForm({ forced_outbound_format: 'gemini' })
      ).success
    ).toBe(true)
    expect(
      channelFormSchema.safeParse(
        createValidForm({
          type: CHANNEL_TYPE_NEW_API,
          base_url: 'https://new-api.example',
          forced_outbound_format: 'claude',
        })
      ).success
    ).toBe(true)

    const unsupported = channelFormSchema.safeParse(
      createValidForm({
        type: 14,
        forced_outbound_format: 'openai_responses',
      })
    )
    expect(unsupported.success).toBe(false)
    if (!unsupported.success) {
      expect(unsupported.error.issues).toContainEqual(
        expect.objectContaining({ path: ['forced_outbound_format'] })
      )
    }
  })

  test('keeps passthrough conflicts valid and groups format errors under advanced settings', () => {
    const result = channelFormSchema.safeParse(
      createValidForm({
        forced_outbound_format: 'openai',
        pass_through_body_enabled: true,
      })
    )

    expect(result.success).toBe(true)
    expect(
      hasAdvancedSettingsErrors({
        forced_outbound_format: { type: 'custom', message: 'invalid' },
      })
    ).toBe(true)
  })
})
