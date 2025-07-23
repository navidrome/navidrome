import React, { useState, useEffect } from 'react'
import { useSelector } from 'react-redux'
import { useNotify, useTranslate } from 'react-admin'
import {
  Popover,
  CircularProgress,
  IconButton,
  makeStyles,
  Tooltip,
  Card,
  CardContent,
  CardActions,
  Divider,
  Box,
  Typography,
} from '@material-ui/core'
import { FiActivity } from 'react-icons/fi'
import { BiError } from 'react-icons/bi'
import { VscSync } from 'react-icons/vsc'
import { GiMagnifyingGlass } from 'react-icons/gi'
import subsonic from '../subsonic'
import { useInitialScanStatus } from './useInitialScanStatus'
import { useInterval } from '../common'
import { useScanElapsedTime } from './useScanElapsedTime'
import { formatDuration, formatShortDuration } from '../utils'
import config from '../config'

const useStyles = makeStyles((theme) => ({
  wrapper: {
    position: 'relative',
    color: (props) => (props.up ? null : 'orange'),
  },
  progress: {
    color: theme.palette.primary.light,
    position: 'absolute',
    top: 10,
    left: 10,
    zIndex: 1,
  },
  button: {
    color: 'inherit',
    zIndex: 2,
  },
  counterStatus: {
    minWidth: '20em',
  },
  error: {
    color: theme.palette.error.main,
  },
  card: {
    maxWidth: 'none',
  },
  cardContent: {
    padding: theme.spacing(2, 3),
  },
}))

const getUptime = (serverStart) =>
  formatDuration((Date.now() - serverStart.startTime) / 1000)

const Uptime = () => {
  const serverStart = useSelector((state) => state.activity.serverStart)
  const [uptime, setUptime] = useState(getUptime(serverStart))
  useInterval(() => {
    setUptime(getUptime(serverStart))
  }, 1000)
  return <span>{uptime}</span>
}

const ActivityPanel = () => {
  const serverStart = useSelector((state) => state.activity.serverStart)
  const up = serverStart.startTime
  const scanStatus = useSelector((state) => state.activity.scanStatus)
  const elapsed = useScanElapsedTime(
    scanStatus.scanning,
    scanStatus.elapsedTime,
  )
  const [acknowledgedError, setAcknowledgedError] = useState(null)
  const isErrorVisible =
    scanStatus.error && scanStatus.error !== acknowledgedError
  const classes = useStyles({
    up: up && (!scanStatus.error || !isErrorVisible),
  })
  const translate = useTranslate()
  const notify = useNotify()
  const [anchorEl, setAnchorEl] = useState(null)
  const open = Boolean(anchorEl)
  useInitialScanStatus()

  const handleMenuOpen = (event) => {
    if (scanStatus.error) {
      setAcknowledgedError(scanStatus.error)
    }
    setAnchorEl(event.currentTarget)
  }

  const handleMenuClose = () => setAnchorEl(null)
  const triggerScan = (full) => () => subsonic.startScan({ fullScan: full })

  useEffect(() => {
    if (serverStart.version && serverStart.version !== config.version) {
      notify('ra.notification.new_version', 'info', {}, false, 604800000 * 50)
    }
  }, [serverStart, notify])

  const tooltipTitle = scanStatus.error
    ? `${translate('activity.status')}: ${scanStatus.error}`
    : translate('activity.title')

  const lastScanType = (() => {
    switch (scanStatus.scanType) {
      case 'full':
        return translate('activity.fullScan')
      case 'quick':
        return translate('activity.quickScan')
      default:
        return ''
    }
  })()

  return (
    <div className={classes.wrapper}>
      <Tooltip title={tooltipTitle}>
        <IconButton className={classes.button} onClick={handleMenuOpen}>
          {!up || isErrorVisible ? (
            <BiError data-testid="activity-error-icon" size={'20'} />
          ) : (
            <FiActivity data-testid="activity-ok-icon" size={'20'} />
          )}
        </IconButton>
      </Tooltip>
      {scanStatus.scanning && (
        <CircularProgress size={24} className={classes.progress} />
      )}
      <Popover
        id="panel-activity"
        anchorEl={anchorEl}
        anchorOrigin={{
          vertical: 'bottom',
          horizontal: 'right',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'right',
        }}
        open={open}
        onClose={handleMenuClose}
      >
        <Card className={classes.card}>
          <CardContent className={classes.cardContent}>
            <Box display="flex" className={classes.counterStatus}>
              <Box component="span" flex={2}>
                {translate('activity.serverUptime')}:
              </Box>
              <Box component="span" flex={1}>
                {up ? <Uptime /> : translate('activity.serverDown')}
              </Box>
            </Box>
          </CardContent>
          <Divider />
          <CardContent className={classes.cardContent}>
            <Box display="flex" className={classes.counterStatus}>
              <Box component="span" flex={2}>
                {translate('activity.totalScanned')}:
              </Box>
              <Box component="span" flex={1}>
                {scanStatus.folderCount || '-'}
              </Box>
            </Box>

            <Box display="flex" className={classes.counterStatus} mt={2}>
              <Box component="span" flex={2}>
                {translate('activity.scanType')}:
              </Box>
              <Box component="span" flex={1}>
                {lastScanType}
              </Box>
            </Box>

            <Box display="flex" className={classes.counterStatus} mt={2}>
              <Box component="span" flex={2}>
                {translate('activity.elapsedTime')}:
              </Box>
              <Box component="span" flex={1}>
                {formatShortDuration(elapsed)}
              </Box>
            </Box>

            {scanStatus.error && (
              <Box
                display="flex"
                flexDirection="column"
                mt={2}
                className={classes.error}
              >
                <Typography variant="subtitle2">
                  {translate('activity.status')}:
                </Typography>
                <Typography variant="body2">{scanStatus.error}</Typography>
              </Box>
            )}
          </CardContent>
          <Divider />
          <CardActions>
            <Tooltip title={translate('activity.quickScan')}>
              <IconButton
                onClick={triggerScan(false)}
                disabled={scanStatus.scanning}
              >
                <VscSync />
              </IconButton>
            </Tooltip>
            <Tooltip title={translate('activity.fullScan')}>
              <IconButton
                onClick={triggerScan(true)}
                disabled={scanStatus.scanning}
              >
                <GiMagnifyingGlass />
              </IconButton>
            </Tooltip>
          </CardActions>
        </Card>
      </Popover>
    </div>
  )
}

export default ActivityPanel
