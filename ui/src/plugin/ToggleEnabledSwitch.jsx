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
}))

/**
 * Shared toggle switch for enabling/disabling plugins.
 * Used in both PluginList (compact) and PluginShow (with label).
 *
 * @param {Object} props
 * @param {boolean} [props.showLabel=false] - Whether to show the enable/disable label
 * @param {string} [props.size='small'] - Switch size ('small' or 'medium')
 */
const ToggleEnabledSwitch = ({ showLabel = false, size = 'small' }) => {
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
  const isDisabled = loading || hasError

  const tooltipTitle = useMemo(() => {
    if (hasError) {
      return translate('resources.plugin.actions.disabledDueToError')
    }
    if (!showLabel) {
      return translate(
        record?.enabled
          ? 'resources.plugin.actions.disable'
          : 'resources.plugin.actions.enable',
      )
    }
    return ''
  }, [hasError, showLabel, record?.enabled, translate])

  const switchElement = (
    <Switch
      checked={record?.enabled ?? false}
      onClick={handleClick}
      disabled={isDisabled}
      className={classes.enabledSwitch}
      size={size}
      color="primary"
    />
  )

  if (showLabel) {
    return (
      <Tooltip
        title={tooltipTitle}
        disableHoverListener={!hasError}
        disableFocusListener={!hasError}
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

export default ToggleEnabledSwitch
