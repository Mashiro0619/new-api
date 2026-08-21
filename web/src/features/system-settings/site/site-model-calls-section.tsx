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
import { useQuery } from '@tanstack/react-query'
import { Save } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'

import {
  parseSiteModelCallsConfig,
  serializeSiteModelCallsConfig,
  type SiteModelCallsConfig,
} from '@/features/site-model-calls/config'

import { SettingsSwitchField } from '../components/settings-form-layout'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { getSiteModelCallModels } from '@/features/site-model-calls/api'

type SiteModelCallsSectionProps = {
  initialValue: string
}

export function SiteModelCallsSection({
  initialValue,
}: SiteModelCallsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const initialConfig = useMemo(
    () => parseSiteModelCallsConfig(initialValue),
    [initialValue]
  )
  const [config, setConfig] = useState<SiteModelCallsConfig>(initialConfig)
  const modelsQuery = useQuery({
    queryKey: ['site-model-calls', 'models'],
    queryFn: getSiteModelCallModels,
  })

  useEffect(() => {
    setConfig(initialConfig)
  }, [initialConfig])

  const options = useMemo(() => {
    const values = new Set([
      ...(modelsQuery.data?.data ?? []),
      ...config.models,
    ])
    return [...values]
      .sort((a, b) => a.localeCompare(b))
      .map((value) => ({ label: value, value }))
  }, [config.models, modelsQuery.data?.data])

  const handleSave = async () => {
    try {
      await updateOption.mutateAsync({
        key: 'AllSiteModelCalls',
        value: serializeSiteModelCallsConfig(config),
      })
    } catch {
      // The mutation hook displays the server error.
    }
  }

  return (
    <SettingsSection title={t('All-site model calls')}>
      <div className='space-y-4'>
        <SettingsSwitchField
          checked={config.enabled}
          onCheckedChange={(enabled) => setConfig((prev) => ({ ...prev, enabled }))}
          label={t('Open all-site model calls')}
          description={t('Allow regular users to view cumulative model call counts.')}
        />
        <div className='space-y-2'>
          <label htmlFor='site-model-calls-models' className='text-sm font-medium'>
            {t('Displayed models')}
          </label>
          <p className='text-muted-foreground text-xs'>
            {t('Leave empty to display all models.')}
          </p>
          <MultiSelect
            options={options}
            selected={config.models}
            onChange={(models) => setConfig((prev) => ({ ...prev, models }))}
            placeholder={t('Select models')}
            emptyText={t('No models found')}
            maxVisibleChips={6}
            id='site-model-calls-models'
          />
        </div>
        <div className='flex justify-end'>
          <Button onClick={handleSave} disabled={updateOption.isPending}>
            <Save className='mr-2 h-4 w-4' />
            {updateOption.isPending ? t('Saving...') : t('Save Settings')}
          </Button>
        </div>
      </div>
    </SettingsSection>
  )
}
