import React, { useState, useCallback, useMemo } from 'react'
import {
  ShowContextProvider,
  useShowController,
  useShowContext,
  useTranslate,
  useUpdate,
  useNotify,
  useRefresh,
  Title as RaTitle,
  Loading,
} from 'react-admin'
import { Box, useMediaQuery } from '@material-ui/core'
import Alert from '@material-ui/lab/Alert'
import { Title } from '../common'
import { usePluginShowStyles } from './styles.js'
import { ErrorSection } from './ErrorSection'
import { StatusCard } from './StatusCard'
import { InfoCard } from './InfoCard'
import { ManifestSection } from './ManifestSection'
import { ConfigCard } from './ConfigCard'

// Main show layout component
const PluginShowLayout = () => {
  const { record, isPending, error } = useShowContext()
  const classes = usePluginShowStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const isSmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))

  const [configPairs, setConfigPairs] = useState([])
  const [isDirty, setIsDirty] = useState(false)
  const [configInitialized, setConfigInitialized] = useState(false)

  // Convert JSON config to key-value pairs
  const jsonToPairs = useCallback((jsonString) => {
    if (!jsonString || jsonString.trim() === '') return []
    try {
      const obj = JSON.parse(jsonString)
      return Object.entries(obj).map(([key, value]) => ({
        key,
        value: typeof value === 'string' ? value : JSON.stringify(value),
      }))
    } catch {
      return []
    }
  }, [])

  // Convert key-value pairs to JSON config
  const pairsToJson = useCallback((pairs) => {
    if (pairs.length === 0) return ''
    const obj = {}
    pairs.forEach((pair) => {
      if (pair.key.trim()) {
        // Try to parse value as JSON, otherwise use as string
        try {
          obj[pair.key] = JSON.parse(pair.value)
        } catch {
          obj[pair.key] = pair.value
        }
      }
    })
    return JSON.stringify(obj)
  }, [])

  // Initialize config when record loads
  React.useEffect(() => {
    if (record && !configInitialized) {
      setConfigPairs(jsonToPairs(record.config || ''))
      setConfigInitialized(true)
    }
  }, [record, configInitialized, jsonToPairs])

  const handleConfigPairsChange = useCallback(
    (newPairs) => {
      setConfigPairs(newPairs)
      const newJson = pairsToJson(newPairs)
      const originalJson = record?.config || ''
      setIsDirty(newJson !== originalJson)
    },
    [record?.config, pairsToJson],
  )

  const [updatePlugin, { loading }] = useUpdate(
    'plugin',
    record?.id,
    {},
    record,
    {
      undoable: false,
      onSuccess: () => {
        refresh()
        setIsDirty(false)
        setConfigInitialized(false) // Reset to reinitialize from server
        notify('resources.plugin.notifications.updated', 'info')
      },
      onFailure: (err) => {
        notify(
          err?.message || 'resources.plugin.notifications.error',
          'warning',
        )
      },
    },
  )

  const handleSaveConfig = useCallback(() => {
    if (!record) return
    const config = pairsToJson(configPairs)
    updatePlugin('plugin', record.id, { config }, record)
  }, [updatePlugin, record, configPairs, pairsToJson])

  // Parse manifest
  const { manifest, manifestJson } = useMemo(() => {
    if (!record?.manifest) return { manifest: null, manifestJson: '' }
    try {
      const parsed = JSON.parse(record.manifest)
      return { manifest: parsed, manifestJson: JSON.stringify(parsed, null, 2) }
    } catch {
      return { manifest: null, manifestJson: record.manifest }
    }
  }, [record?.manifest])

  // Handle loading state
  if (isPending) {
    return <Loading />
  }

  // Handle error state
  if (error) {
    return (
      <Alert severity="error">{translate('ra.notification.http_error')}</Alert>
    )
  }

  // Handle missing record
  if (!record) {
    return null
  }

  return (
    <>
      <RaTitle
        title={
          <Title
            subTitle={`${translate('resources.plugin.name', { smart_count: 1 })} "${record.id}"`}
          />
        }
      />
      <Box className={classes.root}>
        <ErrorSection error={record.lastError} translate={translate} />

        <StatusCard classes={classes} translate={translate} />

        <InfoCard
          record={record}
          manifest={manifest}
          classes={classes}
          translate={translate}
          isSmall={isSmall}
        />

        <ManifestSection
          manifestJson={manifestJson}
          classes={classes}
          translate={translate}
        />

        <ConfigCard
          configPairs={configPairs}
          onConfigPairsChange={handleConfigPairsChange}
          isDirty={isDirty}
          loading={loading}
          classes={classes}
          translate={translate}
          onSave={handleSaveConfig}
        />
      </Box>
    </>
  )
}

const PluginShow = (props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <PluginShowLayout />
    </ShowContextProvider>
  )
}

export default PluginShow
