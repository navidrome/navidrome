import React, { useMemo, useState, useCallback } from 'react'
import {
  Button,
  Datagrid,
  TextField,
  TopToolbar,
  useNotify,
  useRecordContext,
  useRefresh,
  useTranslate,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { useMediaQuery, Tooltip, Chip, Typography } from '@material-ui/core'
import { MdError, MdRefresh } from 'react-icons/md'
import { List, DateField, SimpleList, useResourceRefresh } from '../common'
import { httpClient } from '../dataProvider'
import ToggleEnabledSwitch from './ToggleEnabledSwitch'

const useStyles = makeStyles((theme) => ({
  errorIcon: {
    color: theme.palette.error.main,
    marginRight: theme.spacing(0.5),
    verticalAlign: 'middle',
  },
  errorChip: {
    backgroundColor: theme.palette.error.light,
    color: theme.palette.error.contrastText,
  },
}))

const useManifest = () => {
  const record = useRecordContext()
  return useMemo(() => {
    if (!record?.manifest) return null
    try {
      return JSON.parse(record.manifest)
    } catch {
      return null
    }
  }, [record?.manifest])
}

const EnabledOrErrorField = () => {
  const record = useRecordContext()
  const translate = useTranslate()
  const classes = useStyles()
  const manifest = useManifest()

  if (record.lastError) {
    return (
      <Tooltip title={record.lastError}>
        <Chip
          size="small"
          icon={<MdError className={classes.errorIcon} />}
          label={translate('resources.plugin.fields.hasError')}
          className={classes.errorChip}
        />
      </Tooltip>
    )
  }

  return <ToggleEnabledSwitch source={'enabled'} manifest={manifest} />
}

const ManifestField = ({ source }) => {
  const manifest = useManifest()

  if (!manifest) {
    return <Typography variant="body2">-</Typography>
  }

  return <Typography variant="body2">{manifest[source] || '-'}</Typography>
}

const PluginListActions = () => {
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const [loading, setLoading] = useState(false)

  const handleRescan = useCallback(() => {
    setLoading(true)
    httpClient('/api/plugin/rescan', { method: 'POST' })
      .then(() => {
        refresh()
      })
      .catch((error) => {
        notify(error.message || 'ra.page.error', { type: 'warning' })
      })
      .finally(() => {
        setLoading(false)
      })
  }, [notify, refresh])

  return (
    <TopToolbar>
      <Button
        onClick={handleRescan}
        disabled={loading}
        label={translate('resources.plugin.actions.rescan')}
        data-testid="rescan-button"
      >
        <MdRefresh />
      </Button>
    </TopToolbar>
  )
}

const PluginList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const translate = useTranslate()
  useResourceRefresh('plugin')

  return (
    <List
      {...props}
      sort={{ field: 'id', order: 'ASC' }}
      exporter={false}
      bulkActionButtons={false}
      actions={<PluginListActions />}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(record) => record.id}
          secondaryText={(record) => {
            try {
              const manifest = JSON.parse(record.manifest)
              return manifest.description || ''
            } catch {
              return ''
            }
          }}
          tertiaryText={(record) =>
            record.enabled
              ? translate('resources.plugin.status.enabled')
              : translate('resources.plugin.status.disabled')
          }
          linkType="show"
        />
      ) : (
        <Datagrid rowClick="show">
          <TextField source="id" />
          <ManifestField source="name" />
          {!isXsmall && <ManifestField source="description" />}
          <ManifestField source="version" />
          <EnabledOrErrorField source={'enabled'} />
          <DateField source="updatedAt" sortByOrder={'DESC'} />
        </Datagrid>
      )}
    </List>
  )
}

export default PluginList
