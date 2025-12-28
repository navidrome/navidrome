import React, { useState, useCallback } from 'react'
import {
  Show,
  SimpleShowLayout,
  TextField,
  useTranslate,
  useUpdate,
  useNotify,
  useRefresh,
  useRecordContext,
  Toolbar,
  SaveButton,
} from 'react-admin'
import {
  Typography,
  Box,
  Switch,
  FormControlLabel,
  Card,
  CardContent,
  TextField as MuiTextField,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
  Tooltip,
  Link,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { MdExpandMore, MdError } from 'react-icons/md'
import { Title, DateField } from '../common'
import { validateJson } from './jsonValidation'

const useStyles = makeStyles((theme) => ({
  root: {
    padding: theme.spacing(2),
    maxWidth: 900,
  },
  section: {
    marginBottom: theme.spacing(3),
  },
  sectionTitle: {
    marginBottom: theme.spacing(1),
    fontWeight: 600,
  },
  errorBox: {
    backgroundColor: theme.palette.error.light,
    color: theme.palette.error.contrastText,
    padding: theme.spacing(2),
    borderRadius: theme.shape.borderRadius,
    marginBottom: theme.spacing(2),
    display: 'flex',
    alignItems: 'flex-start',
    gap: theme.spacing(1),
  },
  errorIcon: {
    marginTop: 2,
  },
  manifestBox: {
    backgroundColor:
      theme.palette.type === 'dark'
        ? theme.palette.grey[900]
        : theme.palette.grey[100],
    padding: theme.spacing(2),
    borderRadius: theme.shape.borderRadius,
    fontFamily: 'monospace',
    fontSize: '0.85rem',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
    overflow: 'auto',
    maxHeight: 400,
  },
  configInput: {
    fontFamily: 'monospace',
    fontSize: '0.85rem',
  },
  toolbar: {
    display: 'flex',
    justifyContent: 'flex-start',
    paddingLeft: 0,
    paddingRight: 0,
    marginTop: theme.spacing(2),
  },
  infoGrid: {
    display: 'grid',
    gridTemplateColumns: 'auto 1fr',
    gap: theme.spacing(1, 2),
    '& dt': {
      fontWeight: 500,
      color: theme.palette.text.secondary,
    },
    '& dd': {
      margin: 0,
    },
  },
  pathField: {
    fontFamily: 'monospace',
    fontSize: '0.85rem',
    wordBreak: 'break-all',
  },
  permissionsContainer: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(0.5),
  },
  permissionChip: {
    fontSize: '0.75rem',
  },
  tooltipContent: {
    '& code': {
      fontFamily: 'monospace',
      fontSize: '0.8em',
      backgroundColor: 'rgba(255,255,255,0.1)',
      padding: '1px 4px',
      borderRadius: 2,
    },
  },
}))

const PermissionChip = ({ label, permission }) => {
  const classes = useStyles()

  if (!permission) return null

  const hasHosts = permission.allowedHosts?.length > 0
  const tooltipContent = (
    <Box className={classes.tooltipContent}>
      {permission.reason && <Typography variant="body2">{permission.reason}</Typography>}
      {hasHosts && (
        <Box mt={permission.reason ? 0.5 : 0}>
          <Typography variant="caption" component="div">
            Allowed hosts: {permission.allowedHosts.map((host, i) => (
              <span key={host}>{i > 0 && ', '}<code>{host}</code></span>
            ))}
          </Typography>
        </Box>
      )}
    </Box>
  )

  const hasTooltip = permission.reason || hasHosts

  const chip = (
    <Chip
      size="small"
      label={label}
      className={classes.permissionChip}
    />
  )

  return hasTooltip ? (
    <Tooltip title={tooltipContent} arrow>
      {chip}
    </Tooltip>
  ) : chip
}

const PluginTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.plugin.name', { smart_count: 1 })
  return (
    <Title subTitle={`${resourceName} ${record ? `"${record.id}"` : ''}`} />
  )
}

const PluginShowContent = () => {
  const record = useRecordContext()
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()

  const [config, setConfig] = useState(record?.config || '')
  const [configError, setConfigError] = useState(null)
  const [isDirty, setIsDirty] = useState(false)

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
        notify('resources.plugin.notifications.updated', 'info')
      },
      onFailure: (error) => {
        notify(
          error?.message || 'resources.plugin.notifications.error',
          'warning',
        )
      },
    },
  )

  const handleToggleEnabled = useCallback(() => {
    updatePlugin('plugin', record.id, { enabled: !record.enabled }, record)
  }, [updatePlugin, record])

  const handleConfigChange = useCallback(
    (e) => {
      const value = e.target.value
      setConfig(value)
      setIsDirty(value !== (record?.config || ''))

      if (value === '') {
        setConfigError(null)
      } else {
        const validation = validateJson(value)
        setConfigError(validation.error)
      }
    },
    [record?.config],
  )

  const handleSaveConfig = useCallback(() => {
    if (configError) {
      notify('resources.plugin.validation.invalidJson', 'warning')
      return
    }
    updatePlugin('plugin', record.id, { config }, record)
  }, [updatePlugin, record, config, configError, notify])

  if (!record) {
    return null
  }

  let manifest = null
  let manifestJson = ''
  try {
    manifest = JSON.parse(record.manifest)
    manifestJson = JSON.stringify(manifest, null, 2)
  } catch {
    manifestJson = record.manifest
  }

  return (
    <Box className={classes.root}>
      {/* Error Section */}
      {record.lastError && (
        <Box className={classes.errorBox}>
          <MdError size={20} className={classes.errorIcon} />
          <Box>
            <Typography variant="subtitle2">
              {translate('resources.plugin.fields.lastError')}
            </Typography>
            <Typography variant="body2">{record.lastError}</Typography>
          </Box>
        </Box>
      )}

      {/* Status - Enable/Disable Switch Only */}
      <Card className={classes.section}>
        <CardContent>
          <Typography variant="h6" className={classes.sectionTitle}>
            {translate('resources.plugin.sections.status')}
          </Typography>
          <FormControlLabel
            control={
              <Switch
                checked={record.enabled}
                onChange={handleToggleEnabled}
                disabled={loading}
                color="primary"
              />
            }
            label={translate(
              record.enabled
                ? 'resources.plugin.actions.disable'
                : 'resources.plugin.actions.enable',
            )}
          />
        </CardContent>
      </Card>

      {/* Plugin Info */}
      <Card className={classes.section}>
        <CardContent>
          <Typography variant="h6" className={classes.sectionTitle}>
            {translate('resources.plugin.sections.info')}
          </Typography>
          <dl className={classes.infoGrid}>
            <dt>{translate('resources.plugin.fields.id')}</dt>
            <dd>{record.id}</dd>

            {manifest?.name && (
              <>
                <dt>{translate('resources.plugin.fields.name')}</dt>
                <dd>{manifest.name}</dd>
              </>
            )}

            {manifest?.version && (
              <>
                <dt>{translate('resources.plugin.fields.version')}</dt>
                <dd>{manifest.version}</dd>
              </>
            )}

            {manifest?.description && (
              <>
                <dt>{translate('resources.plugin.fields.description')}</dt>
                <dd>{manifest.description}</dd>
              </>
            )}

            {manifest?.author && (
              <>
                <dt>{translate('resources.plugin.fields.author')}</dt>
                <dd>{manifest.author}</dd>
              </>
            )}

            {manifest?.website && (
              <>
                <dt>{translate('resources.plugin.fields.website')}</dt>
                <dd>
                  <Link
                    href={manifest.website}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {manifest.website}
                  </Link>
                </dd>
              </>
            )}

            {manifest?.permissions && (
              <>
                <dt>{translate('resources.plugin.fields.permissions')}</dt>
                <dd className={classes.permissionsContainer}>
                  <PermissionChip label="HTTP" permission={manifest.permissions.http} />
                  <PermissionChip label="Subsonic API" permission={manifest.permissions.subsonicapi} />
                  <PermissionChip label="Scheduler" permission={manifest.permissions.scheduler} />
                  <PermissionChip label="WebSocket" permission={manifest.permissions.websocket} />
                  <PermissionChip label="Artwork" permission={manifest.permissions.artwork} />
                  <PermissionChip label="Cache" permission={manifest.permissions.cache} />
                </dd>
              </>
            )}

            <dt>{translate('resources.plugin.fields.path')}</dt>
            <dd className={classes.pathField}>{record.path}</dd>

            <dt>{translate('resources.plugin.fields.updatedAt')}</dt>
            <dd>
              <DateField record={record} source="updatedAt" showTime />
            </dd>

            <dt>{translate('resources.plugin.fields.createdAt')}</dt>
            <dd>
              <DateField record={record} source="createdAt" showTime />
            </dd>
          </dl>
        </CardContent>
      </Card>

      {/* Manifest (Collapsible) */}
      <Accordion className={classes.section}>
        <AccordionSummary expandIcon={<MdExpandMore />}>
          <Typography variant="h6">
            {translate('resources.plugin.sections.manifest')}
          </Typography>
        </AccordionSummary>
        <AccordionDetails>
          <Box className={classes.manifestBox} width="100%">
            {manifestJson}
          </Box>
        </AccordionDetails>
      </Accordion>

      {/* Configuration */}
      <Card className={classes.section}>
        <CardContent>
          <Typography variant="h6" className={classes.sectionTitle}>
            {translate('resources.plugin.sections.configuration')}
          </Typography>
          <Typography variant="body2" color="textSecondary" gutterBottom>
            {translate('resources.plugin.messages.configHelp')}
          </Typography>
          <MuiTextField
            multiline
            fullWidth
            minRows={4}
            maxRows={15}
            variant="outlined"
            value={config}
            onChange={handleConfigChange}
            error={!!configError}
            helperText={configError}
            placeholder="{}"
            InputProps={{
              className: classes.configInput,
            }}
          />
          <Toolbar className={classes.toolbar}>
            <SaveButton
              handleSubmitWithRedirect={handleSaveConfig}
              disabled={!isDirty || !!configError || loading}
              saving={loading}
            />
          </Toolbar>
        </CardContent>
      </Card>
    </Box>
  )
}

const PluginShow = (props) => {
  return (
    <Show title={<PluginTitle />} actions={false} {...props}>
      <SimpleShowLayout>
        <PluginShowContent />
      </SimpleShowLayout>
    </Show>
  )
}

export default PluginShow
