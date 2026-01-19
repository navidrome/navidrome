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
import { Box, useMediaQuery, Button } from '@material-ui/core'
import { MdSave } from 'react-icons/md'
import Alert from '@material-ui/lab/Alert'
import { Title, useResourceRefresh } from '../common'
import { usePluginShowStyles } from './styles.js'
import { ErrorSection } from './ErrorSection'
import { StatusCard } from './StatusCard'
import { InfoCard } from './InfoCard'
import { ManifestSection } from './ManifestSection'
import { ConfigCard } from './ConfigCard'
import { UsersPermissionCard } from './UsersPermissionCard'
import { LibraryPermissionCard } from './LibraryPermissionCard'

// Main show layout component
const PluginShowLayout = () => {
  const { record, isPending, error } = useShowContext()
  const classes = usePluginShowStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const isSmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  useResourceRefresh('plugin')

  const [configData, setConfigData] = useState({})
  const [configErrors, setConfigErrors] = useState([])
  const [isDirty, setIsDirty] = useState(false)
  const [lastRecordConfig, setLastRecordConfig] = useState(null)
  const [isConfigInitialized, setIsConfigInitialized] = useState(false)

  // Users permission state
  const [selectedUsers, setSelectedUsers] = useState([])
  const [allUsers, setAllUsers] = useState(false)
  const [lastRecordUsers, setLastRecordUsers] = useState(null)
  const [lastRecordAllUsers, setLastRecordAllUsers] = useState(null)

  // Libraries permission state
  const [selectedLibraries, setSelectedLibraries] = useState([])
  const [allLibraries, setAllLibraries] = useState(false)
  const [lastRecordLibraries, setLastRecordLibraries] = useState(null)
  const [lastRecordAllLibraries, setLastRecordAllLibraries] = useState(null)

  // Parse JSON config to object
  const jsonToObject = useCallback((jsonString) => {
    if (!jsonString || jsonString.trim() === '') return {}
    try {
      return JSON.parse(jsonString)
    } catch {
      return {}
    }
  }, [])

  // Initialize/update config when record loads or changes (e.g., from SSE refresh)
  React.useEffect(() => {
    const recordConfig = record?.config || ''
    if (record && recordConfig !== lastRecordConfig && !isDirty) {
      setConfigData(jsonToObject(recordConfig))
      setLastRecordConfig(recordConfig)
      // Reset initialization flag - AJV will apply defaults on first render
      setIsConfigInitialized(false)
    }
  }, [record, lastRecordConfig, isDirty, jsonToObject])

  // Initialize/update users permission state when record loads or changes
  React.useEffect(() => {
    if (record && !isDirty) {
      const recordUsers = record.users || ''
      const recordAllUsers = record.allUsers || false

      if (
        recordUsers !== lastRecordUsers ||
        recordAllUsers !== lastRecordAllUsers
      ) {
        try {
          setSelectedUsers(recordUsers ? JSON.parse(recordUsers) : [])
        } catch {
          setSelectedUsers([])
        }
        setAllUsers(recordAllUsers)
        setLastRecordUsers(recordUsers)
        setLastRecordAllUsers(recordAllUsers)
      }
    }
  }, [record, lastRecordUsers, lastRecordAllUsers, isDirty])

  // Initialize/update libraries permission state when record loads or changes
  React.useEffect(() => {
    if (record && !isDirty) {
      const recordLibraries = record.libraries || ''
      const recordAllLibraries = record.allLibraries || false

      if (
        recordLibraries !== lastRecordLibraries ||
        recordAllLibraries !== lastRecordAllLibraries
      ) {
        try {
          setSelectedLibraries(
            recordLibraries ? JSON.parse(recordLibraries) : [],
          )
        } catch {
          setSelectedLibraries([])
        }
        setAllLibraries(recordAllLibraries)
        setLastRecordLibraries(recordLibraries)
        setLastRecordAllLibraries(recordAllLibraries)
      }
    }
  }, [record, lastRecordLibraries, lastRecordAllLibraries, isDirty])

  const handleConfigDataChange = useCallback(
    (newData, errors) => {
      setConfigData(newData)
      setConfigErrors(errors || [])
      // Skip marking dirty on initial onChange (when AJV applies defaults)
      if (isConfigInitialized) {
        setIsDirty(true)
      } else {
        setIsConfigInitialized(true)
      }
    },
    [isConfigInitialized],
  )

  const handleSelectedUsersChange = useCallback((newSelectedUsers) => {
    setSelectedUsers(newSelectedUsers)
    setIsDirty(true)
  }, [])

  const handleAllUsersChange = useCallback((newAllUsers) => {
    setAllUsers(newAllUsers)
    setIsDirty(true)
  }, [])

  const handleSelectedLibrariesChange = useCallback((newSelectedLibraries) => {
    setSelectedLibraries(newSelectedLibraries)
    setIsDirty(true)
  }, [])

  const handleAllLibrariesChange = useCallback((newAllLibraries) => {
    setAllLibraries(newAllLibraries)
    setIsDirty(true)
  }, [])

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
        setLastRecordConfig(null) // Reset to reinitialize from server
        setLastRecordUsers(null)
        setLastRecordAllUsers(null)
        setLastRecordLibraries(null)
        setLastRecordAllLibraries(null)
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
    const parsedManifest = record.manifest ? JSON.parse(record.manifest) : null
    const data = {}

    // Only include config if the plugin has a config schema
    if (parsedManifest?.config?.schema) {
      data.config =
        Object.keys(configData).length > 0 ? JSON.stringify(configData) : ''
    }

    // Include users data if users permission is present
    if (parsedManifest?.permissions?.users) {
      data.users = JSON.stringify(selectedUsers)
      data.allUsers = allUsers
    }

    // Include libraries data if library permission is present
    if (parsedManifest?.permissions?.library) {
      data.libraries = JSON.stringify(selectedLibraries)
      data.allLibraries = allLibraries
    }

    updatePlugin('plugin', record.id, data, record)
  }, [
    updatePlugin,
    record,
    configData,
    selectedUsers,
    allUsers,
    selectedLibraries,
    allLibraries,
  ])

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

        <StatusCard
          classes={classes}
          translate={translate}
          manifest={manifest}
        />

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
          manifest={manifest}
          configData={configData}
          onConfigDataChange={handleConfigDataChange}
          classes={classes}
          translate={translate}
        />

        <UsersPermissionCard
          manifest={manifest}
          classes={classes}
          selectedUsers={selectedUsers}
          allUsers={allUsers}
          onSelectedUsersChange={handleSelectedUsersChange}
          onAllUsersChange={handleAllUsersChange}
        />

        <LibraryPermissionCard
          manifest={manifest}
          classes={classes}
          selectedLibraries={selectedLibraries}
          allLibraries={allLibraries}
          onSelectedLibrariesChange={handleSelectedLibrariesChange}
          onAllLibrariesChange={handleAllLibrariesChange}
        />

        <Box display="flex" justifyContent="flex-end">
          <Button
            variant="contained"
            color="primary"
            startIcon={<MdSave />}
            onClick={handleSaveConfig}
            disabled={!isDirty || loading || configErrors.length > 0}
            className={classes.saveButton}
          >
            {translate('ra.action.save')}
          </Button>
        </Box>
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
