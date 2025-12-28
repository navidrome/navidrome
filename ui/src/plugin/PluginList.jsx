import React, { useMemo } from 'react'
import {
  Datagrid,
  TextField,
  useRecordContext,
  useTranslate,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { useMediaQuery, Tooltip, Chip, Typography } from '@material-ui/core'
import { MdError } from 'react-icons/md'
import { List, DateField, SimpleList } from '../common'
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

const ManifestField = ({ source }) => {
  const manifest = useManifest()

  if (!manifest) {
    return <Typography variant="body2">-</Typography>
  }

  return <Typography variant="body2">{manifest[source] || '-'}</Typography>
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
          <ManifestField source="name" />
          {!isXsmall && <ManifestField source="description" />}
          <ManifestField source="version" />
          <ToggleEnabledSwitch source={'enabled'} />
          <ErrorIndicator source="lastError" />
          <DateField source="updatedAt" sortByOrder={'DESC'} />
        </Datagrid>
      )}
    </List>
  )
}

export default PluginList
