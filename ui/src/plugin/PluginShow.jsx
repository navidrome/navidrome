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
  Grid,
  useMediaQuery,
  Button,
  ClickAwayListener,
} from '@material-ui/core'
import Alert from '@material-ui/lab/Alert'
import { makeStyles } from '@material-ui/core/styles'
import { MdExpandMore, MdSave } from 'react-icons/md'
import { Title, DateField } from '../common'
import { validateJson } from './jsonValidation'

const useStyles = makeStyles(
  (theme) => ({
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
    saveButton: {
      marginTop: theme.spacing(2),
    },
    infoGrid: {
      '& .MuiGrid-item': {
        paddingTop: theme.spacing(0.5),
        paddingBottom: theme.spacing(0.5),
      },
    },
    infoLabel: {
      fontWeight: 500,
      color: theme.palette.text.secondary,
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
  }),
  { name: 'NDPluginShow' },
)

// Helper component for permission chips with clickable persistent tooltips
const PermissionChip = ({ label, permission, classes }) => {
  const [open, setOpen] = React.useState(false)

  if (!permission) return null

  const hasHosts = permission.allowedHosts?.length > 0
  const hasTooltip = permission.reason || hasHosts

  const handleClick = () => {
    if (hasTooltip) {
      setOpen((prev) => !prev)
    }
  }

  const handleClose = () => {
    setOpen(false)
  }

  const tooltipContent = (
    <Box className={classes.tooltipContent}>
      {permission.reason && (
        <Typography variant="body2">{permission.reason}</Typography>
      )}
      {hasHosts && (
        <Box mt={permission.reason ? 0.5 : 0}>
          <Typography variant="caption" component="div">
            Allowed hosts:{' '}
            {permission.allowedHosts.map((host, i) => (
              <span key={host}>
                {i > 0 && ', '}
                <code>{host}</code>
              </span>
            ))}
          </Typography>
        </Box>
      )}
    </Box>
  )

  const chip = (
    <Chip
      size="small"
      label={label}
      className={classes.permissionChip}
      onClick={hasTooltip ? handleClick : undefined}
      clickable={hasTooltip}
    />
  )

  if (!hasTooltip) {
    return chip
  }

  return (
    <ClickAwayListener onClickAway={handleClose}>
      <div>
        <Tooltip
          title={tooltipContent}
          arrow
          open={open}
          disableFocusListener
          disableHoverListener
          disableTouchListener
          PopperProps={{
            disablePortal: true,
          }}
        >
          {chip}
        </Tooltip>
      </div>
    </ClickAwayListener>
  )
}

// Info row component for responsive grid
const InfoRow = ({ label, children, classes, isSmall }) => (
  <>
    <Grid item xs={12} sm={4}>
      <Typography
        variant="body2"
        className={classes.infoLabel}
        component={isSmall ? 'div' : 'span'}
      >
        {label}
      </Typography>
    </Grid>
    <Grid item xs={12} sm={8}>
      <Typography variant="body2" component="div">
        {children}
      </Typography>
    </Grid>
  </>
)

// Error display section
const ErrorSection = ({ error, translate }) => {
  if (!error) return null

  return (
    <Alert severity="error" style={{ marginBottom: 16 }}>
      <Typography variant="subtitle2">
        {translate('resources.plugin.fields.lastError')}
      </Typography>
      <Typography variant="body2">{error}</Typography>
    </Alert>
  )
}

// Status card with enable/disable toggle
const StatusCard = ({
  record,
  classes,
  translate,
  onToggle,
  loading,
  hasError,
}) => {
  const isDisabled = loading || hasError

  return (
    <Card className={classes.section}>
      <CardContent>
        <Typography variant="h6" className={classes.sectionTitle}>
          {translate('resources.plugin.sections.status')}
        </Typography>
        <Tooltip
          title={
            hasError
              ? translate('resources.plugin.actions.disabledDueToError')
              : ''
          }
          disableHoverListener={!hasError}
        >
          <FormControlLabel
            control={
              <Switch
                checked={record.enabled}
                onChange={onToggle}
                disabled={isDisabled}
                color="primary"
              />
            }
            label={translate(
              record.enabled
                ? 'resources.plugin.actions.disable'
                : 'resources.plugin.actions.enable',
            )}
          />
        </Tooltip>
      </CardContent>
    </Card>
  )
}

// Plugin information card
const InfoCard = ({ record, manifest, classes, translate, isSmall }) => (
  <Card className={classes.section}>
    <CardContent>
      <Typography variant="h6" className={classes.sectionTitle}>
        {translate('resources.plugin.sections.info')}
      </Typography>
      <Grid container spacing={1} className={classes.infoGrid}>
        <InfoRow
          label={translate('resources.plugin.fields.id')}
          classes={classes}
          isSmall={isSmall}
        >
          {record.id}
        </InfoRow>

        {manifest?.name && (
          <InfoRow
            label={translate('resources.plugin.fields.name')}
            classes={classes}
            isSmall={isSmall}
          >
            {manifest.name}
          </InfoRow>
        )}

        {manifest?.version && (
          <InfoRow
            label={translate('resources.plugin.fields.version')}
            classes={classes}
            isSmall={isSmall}
          >
            {manifest.version}
          </InfoRow>
        )}

        {manifest?.description && (
          <InfoRow
            label={translate('resources.plugin.fields.description')}
            classes={classes}
            isSmall={isSmall}
          >
            {manifest.description}
          </InfoRow>
        )}

        {manifest?.author && (
          <InfoRow
            label={translate('resources.plugin.fields.author')}
            classes={classes}
            isSmall={isSmall}
          >
            {manifest.author}
          </InfoRow>
        )}

        {manifest?.website && (
          <InfoRow
            label={translate('resources.plugin.fields.website')}
            classes={classes}
            isSmall={isSmall}
          >
            <Link
              href={manifest.website}
              target="_blank"
              rel="noopener noreferrer"
            >
              {manifest.website}
            </Link>
          </InfoRow>
        )}

        {manifest?.permissions && (
          <InfoRow
            label={translate('resources.plugin.fields.permissions')}
            classes={classes}
            isSmall={isSmall}
          >
            <Box className={classes.permissionsContainer}>
              <PermissionChip
                label="HTTP"
                permission={manifest.permissions.http}
                classes={classes}
              />
              <PermissionChip
                label="Subsonic API"
                permission={manifest.permissions.subsonicapi}
                classes={classes}
              />
              <PermissionChip
                label="Scheduler"
                permission={manifest.permissions.scheduler}
                classes={classes}
              />
              <PermissionChip
                label="WebSocket"
                permission={manifest.permissions.websocket}
                classes={classes}
              />
              <PermissionChip
                label="Artwork"
                permission={manifest.permissions.artwork}
                classes={classes}
              />
              <PermissionChip
                label="Cache"
                permission={manifest.permissions.cache}
                classes={classes}
              />
            </Box>
            <Typography
              variant="caption"
              color="textSecondary"
              style={{ marginTop: 4, display: 'block' }}
            >
              {translate('resources.plugin.messages.clickPermissions')}
            </Typography>
          </InfoRow>
        )}

        <InfoRow
          label={translate('resources.plugin.fields.path')}
          classes={classes}
          isSmall={isSmall}
        >
          <span className={classes.pathField}>{record.path}</span>
        </InfoRow>

        <InfoRow
          label={translate('resources.plugin.fields.updatedAt')}
          classes={classes}
          isSmall={isSmall}
        >
          <DateField record={record} source="updatedAt" showTime />
        </InfoRow>

        <InfoRow
          label={translate('resources.plugin.fields.createdAt')}
          classes={classes}
          isSmall={isSmall}
        >
          <DateField record={record} source="createdAt" showTime />
        </InfoRow>
      </Grid>
    </CardContent>
  </Card>
)

// Manifest accordion
const ManifestSection = ({ manifestJson, classes, translate }) => (
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
)

// Configuration editor card
const ConfigCard = ({
  config,
  configError,
  isDirty,
  loading,
  classes,
  translate,
  onConfigChange,
  onSave,
}) => (
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
        onChange={onConfigChange}
        error={!!configError}
        helperText={configError}
        placeholder="{}"
        InputProps={{
          className: classes.configInput,
        }}
      />
      <Button
        variant="contained"
        color="primary"
        startIcon={<MdSave />}
        onClick={onSave}
        disabled={!isDirty || !!configError || loading}
        className={classes.saveButton}
      >
        {translate('ra.action.save')}
      </Button>
    </CardContent>
  </Card>
)

// Main show layout component
const PluginShowLayout = () => {
  const { record, isPending, error } = useShowContext()
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const isSmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))

  const [config, setConfig] = useState('')
  const [configError, setConfigError] = useState(null)
  const [isDirty, setIsDirty] = useState(false)
  const [configInitialized, setConfigInitialized] = useState(false)

  // Initialize config when record loads
  React.useEffect(() => {
    if (record && !configInitialized) {
      setConfig(record.config || '')
      setConfigInitialized(true)
    }
  }, [record, configInitialized])

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
      onFailure: (err) => {
        notify(
          err?.message || 'resources.plugin.notifications.error',
          'warning',
        )
      },
    },
  )

  const handleToggleEnabled = useCallback(() => {
    if (!record) return
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
    if (configError || !record) {
      notify('resources.plugin.validation.invalidJson', 'warning')
      return
    }
    updatePlugin('plugin', record.id, { config }, record)
  }, [updatePlugin, record, config, configError, notify])

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
          record={record}
          classes={classes}
          translate={translate}
          onToggle={handleToggleEnabled}
          loading={loading}
          hasError={!!record.lastError}
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
          config={config}
          configError={configError}
          isDirty={isDirty}
          loading={loading}
          classes={classes}
          translate={translate}
          onConfigChange={handleConfigChange}
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
