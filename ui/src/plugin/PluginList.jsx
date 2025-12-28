import React, { useCallback } from 'react'
import {
  Datagrid,
  TextField,
  useUpdate,
  useNotify,
  useRefresh,
  useRecordContext,
  useTranslate,
  FunctionField,
  useResourceContext,
} from 'react-admin'
import Switch from '@material-ui/core/Switch'
import { makeStyles } from '@material-ui/core/styles'
import { useMediaQuery, Tooltip, Chip, Typography } from '@material-ui/core'
import { MdError } from 'react-icons/md'
import { List, DateField, SimpleList } from '../common'

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
  enabledSwitch: {
    '& .MuiSwitch-colorSecondary.Mui-checked': {
      color: theme.palette.success?.main || theme.palette.primary.main,
    },
    '& .MuiSwitch-colorSecondary.Mui-checked + .MuiSwitch-track': {
      backgroundColor:
        theme.palette.success?.main || theme.palette.primary.main,
    },
  },
}))

const ToggleEnabledInput = ({ props }) => {
  const resource = useResourceContext(props)
  const record = useRecordContext(props)
  const notify = useNotify()
  const refresh = useRefresh()
  const translate = useTranslate()
  const classes = useStyles()

  const [toggleEnabled, { loading }] = useUpdate(
    resource,
    record.id,
    { enabled: !record.enabled },
    record,
    {
      undoable: false,
      onSuccess: () => {
        refresh()
        notify(
          record.enabled
            ? 'resources.plugin.notifications.disabled'
            : 'resources.plugin.notifications.enabled',
          'info',
        )
      },
      onFailure: (error) => {
        notify(
          error?.message || 'resources.plugin.notifications.error',
          'warning',
        )
      },
    },
  )

  const handleClick = useCallback(
    (e) => {
      e.stopPropagation()
      toggleEnabled()
    },
    [toggleEnabled],
  )

  return (
    <Tooltip
      title={translate(
        record.enabled
          ? 'resources.plugin.actions.disable'
          : 'resources.plugin.actions.enable',
      )}
    >
      <span>
        <Switch
          checked={record.enabled}
          onClick={handleClick}
          disabled={loading}
          className={classes.enabledSwitch}
          size="small"
        />
      </span>
    </Tooltip>
  )
}

const ErrorIndicator = () => {
  const record = useRecordContext()
  const translate = useTranslate()
  const classes = useStyles()

  if (!record.lastError) {
    return null
  }

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

const ManifestField = ({ source }) => {
  const record = useRecordContext()

  if (!record?.manifest) {
    return null
  }

  try {
    const manifest = JSON.parse(record.manifest)
    return <Typography source>{manifest[source] || '-'}</Typography>
  } catch {
    return <Typography source>-</Typography>
  }
}

const PluginList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const translate = useTranslate()

  return (
    <List {...props} sort={{ field: 'id', order: 'ASC' }} exporter={false}>
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
          {!isXsmall && <ManifestField source="description" />}
          <ManifestField source="version" />
          <ToggleEnabledInput source={'enabled'} />
          <ErrorIndicator source="lastError" />
          <DateField source="updatedAt" sortByOrder={'DESC'} />
        </Datagrid>
      )}
    </List>
  )
}

export default PluginList
