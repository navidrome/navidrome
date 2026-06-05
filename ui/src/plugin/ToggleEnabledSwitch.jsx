import React, { useCallback, useMemo } from 'react'
import {
  useUpdate,
  useNotify,
  useRefresh,
  useRecordContext,
  useTranslate,
  useResourceContext,
} from 'react-admin'
import Switch from '@material-ui/core/Switch'
import { makeStyles } from '@material-ui/core/styles'
import { Tooltip, FormControlLabel } from '@material-ui/core'
import PropTypes from 'prop-types'

const useStyles = makeStyles((theme) => ({
  enabledSwitch: {
    '& .MuiSwitch-colorSecondary.Mui-checked': {
      color: theme.palette.success?.main || theme.palette.primary.main,
    },
    '& .MuiSwitch-colorSecondary.Mui-checked + .MuiSwitch-track': {
      backgroundColor:
        theme.palette.success?.main || theme.palette.primary.main,
    },
  },
  errorSwitch: {
    '& .MuiSwitch-thumb': {
      backgroundColor: theme.palette.warning.main,
    },
    '& .MuiSwitch-track': {
      backgroundColor: theme.palette.warning.light,
      opacity: 0.7,
    },
  },
}))

/**
 * Shared toggle switch for enabling/disabling plugins.
 * Used in both PluginList (compact) and PluginShow (with label).
 *
 * @param {Object} props
 * @param {boolean} [props.showLabel=false] - Whether to show the enable/disable label
 * @param {string} [props.size='small'] - Switch size ('small' or 'medium')
 * @param {Object} [props.manifest=null] - Parsed manifest object for permission checking
 */
const ToggleEnabledSwitch = ({
  showLabel = false,
  size = 'small',
  manifest = null,
}) => {
  const resource = useResourceContext()
  const record = useRecordContext()
  const notify = useNotify()
  const refresh = useRefresh()
  const translate = useTranslate()
  const classes = useStyles()

  const [toggleEnabled, { loading }] = useUpdate(
    resource,
    record?.id,
    { enabled: !record?.enabled },
    record,
    {
      undoable: false,
      onSuccess: () => {
        refresh()
        notify(
          record?.enabled
            ? 'resources.plugin.notifications.disabled'
            : 'resources.plugin.notifications.enabled',
          'info',
        )
      },
      onFailure: (error) => {
        refresh()
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

  const hasError = !!record?.lastError

  // Check if users permission is required but not configured
  const usersPermissionRequired = useMemo(() => {
    if (!manifest?.permissions?.users) return false
    if (record?.allUsers) return false
    // Check if users array is empty or not set
    if (!record?.users) return true
    try {
      const users = JSON.parse(record.users)
      return users.length === 0
    } catch {
      return true
    }
  }, [manifest, record?.allUsers, record?.users])

  // Check if library permission is required but not configured
  const libraryPermissionRequired = useMemo(() => {
    if (!manifest?.permissions?.library) return false
    if (record?.allLibraries) return false
    // Check if libraries array is empty or not set
    if (!record?.libraries) return true
    try {
      const libraries = JSON.parse(record.libraries)
      return libraries.length === 0
    } catch {
      return true
    }
  }, [manifest, record?.allLibraries, record?.libraries])

  const permissionRequired =
    usersPermissionRequired || libraryPermissionRequired
  const isDisabled =
    loading || hasError || (permissionRequired && !record?.enabled)

  const tooltipTitle = useMemo(() => {
    if (hasError) {
      return translate('resources.plugin.actions.disabledDueToError')
    }
    if (usersPermissionRequired && !record?.enabled) {
      return translate('resources.plugin.actions.disabledUsersRequired')
    }
    if (libraryPermissionRequired && !record?.enabled) {
      return translate('resources.plugin.actions.disabledLibrariesRequired')
    }
    if (!showLabel) {
      return translate(
        record?.enabled
          ? 'resources.plugin.actions.disable'
          : 'resources.plugin.actions.enable',
      )
    }
    return ''
  }, [
    hasError,
    usersPermissionRequired,
    libraryPermissionRequired,
    showLabel,
    record?.enabled,
    translate,
  ])

  const switchElement = (
    <Switch
      checked={record?.enabled ?? false}
      onClick={handleClick}
      disabled={isDisabled}
      className={isDisabled ? classes.errorSwitch : classes.enabledSwitch}
      size={size}
      color="primary"
    />
  )

  if (showLabel) {
    const showTooltip = hasError || (permissionRequired && !record?.enabled)
    return (
      <Tooltip
        title={tooltipTitle}
        disableHoverListener={!showTooltip}
        disableFocusListener={!showTooltip}
      >
        <FormControlLabel
          control={switchElement}
          label={translate(
            record?.enabled
              ? 'resources.plugin.actions.disable'
              : 'resources.plugin.actions.enable',
          )}
        />
      </Tooltip>
    )
  }

  return (
    <Tooltip title={tooltipTitle}>
      <span>{switchElement}</span>
    </Tooltip>
  )
}

ToggleEnabledSwitch.propTypes = {
  showLabel: PropTypes.bool,
  size: PropTypes.oneOf(['small', 'medium']),
  manifest: PropTypes.object,
}

export default ToggleEnabledSwitch
