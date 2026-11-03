import React, { useCallback, useState } from 'react'
import IconButton from '@material-ui/core/IconButton'
import ListItemIcon from '@material-ui/core/ListItemIcon'
import ListItemText from '@material-ui/core/ListItemText'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import Tooltip from '@material-ui/core/Tooltip'
import CastIcon from '@material-ui/icons/Cast'
import CastConnectedIcon from '@material-ui/icons/CastConnected'
import CloseIcon from '@material-ui/icons/Close'
import { makeStyles } from '@material-ui/core/styles'
import { useTranslate } from 'react-admin'
import { useCast } from './CastContext'

const useStyles = makeStyles((theme) => ({
  active: {
    color: theme.palette.primary.main,
  },
  deviceItem: {
    opacity: '1 !important',
    fontWeight: 600,
    cursor: 'default',
  },
  menuIcon: {
    minWidth: 32,
  },
}))

const CastButton = ({ className, size, iconClassName }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const [anchorEl, setAnchorEl] = useState(null)
  const {
    castAvailable,
    isCasting,
    isConnecting,
    deviceName,
    requestSession,
    stopCasting,
  } = useCast()

  const handleClick = useCallback(
    (e) => {
      e.stopPropagation()
      if (isCasting) {
        setAnchorEl(e.currentTarget)
      } else {
        requestSession()
      }
    },
    [isCasting, requestSession],
  )

  const handleCloseMenu = useCallback((e) => {
    e?.stopPropagation?.()
    setAnchorEl(null)
  }, [])

  const handleStopCasting = useCallback(
    (e) => {
      e.stopPropagation()
      setAnchorEl(null)
      stopCasting()
    },
    [stopCasting],
  )

  if (!castAvailable) {
    return null
  }

  const connectedTitle = deviceName
    ? translate('player.cast.castingTo', {
        device: deviceName,
        _: `Chromecast · ${deviceName}`,
      })
    : translate('player.cast.connected', { _: 'Chromecast connected' })

  const label = isCasting
    ? connectedTitle
    : isConnecting
      ? translate('player.cast.connecting', {
          _: 'Connecting to Chromecast…',
        })
      : translate('player.cast.castToDevice', { _: 'Cast to Chromecast' })

  const stopLabel = deviceName
    ? translate('player.cast.stopCastingDevice', {
        device: deviceName,
        _: `Stop casting to ${deviceName}`,
      })
    : translate('player.cast.stopCasting', { _: 'Stop casting' })

  const IconComponent = isCasting ? CastConnectedIcon : CastIcon
  const menuOpen = Boolean(anchorEl) && isCasting

  return (
    <>
      <Tooltip title={label}>
        <IconButton
          type="button"
          size={size}
          onClick={handleClick}
          aria-label={label}
          aria-pressed={isCasting}
          aria-haspopup={isCasting ? 'true' : undefined}
          aria-expanded={isCasting ? menuOpen : undefined}
          data-testid="cast-button"
          className={`${className || ''} ${isCasting ? classes.active : ''}`.trim()}
        >
          <IconComponent
            aria-hidden="true"
            fontSize="small"
            className={iconClassName}
          />
        </IconButton>
      </Tooltip>
      {isCasting && (
        <Menu
          anchorEl={anchorEl}
          open={menuOpen}
          onClose={handleCloseMenu}
          anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
          transformOrigin={{ vertical: 'bottom', horizontal: 'center' }}
          getContentAnchorEl={null}
          data-testid="cast-menu"
        >
          <MenuItem disabled className={classes.deviceItem}>
            <ListItemIcon className={classes.menuIcon}>
              <CastConnectedIcon
                fontSize="small"
                color="primary"
                aria-hidden="true"
              />
            </ListItemIcon>
            <ListItemText primary={connectedTitle} />
          </MenuItem>
          <MenuItem onClick={handleStopCasting} data-testid="cast-stop-button">
            <ListItemIcon className={classes.menuIcon}>
              <CloseIcon fontSize="small" aria-hidden="true" />
            </ListItemIcon>
            <ListItemText primary={stopLabel} />
          </MenuItem>
        </Menu>
      )}
    </>
  )
}

export default CastButton
