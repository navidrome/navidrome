import React, { useState, useEffect } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { fetchUtils, useTranslate } from 'react-admin'
import {
  Popover,
  Badge,
  CircularProgress,
  IconButton,
  makeStyles,
  Tooltip,
  Card,
  CardContent,
  CardActions,
  Divider,
  Box,
} from '@material-ui/core'
import { FiActivity } from 'react-icons/fi'
import { VscSync } from 'react-icons/vsc'
import { GiMagnifyingGlass } from 'react-icons/gi'
import subsonic from '../subsonic'
import { scanStatusUpdate } from '../actions'

const useStyles = makeStyles((theme) => ({
  wrapper: {
    position: 'relative',
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
    minWidth: '16em',
  },
}))

const ActivityPanel = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const [anchorEl, setAnchorEl] = useState(null)
  const open = Boolean(anchorEl)
  const scanStatus = useSelector((state) => state.activity.scanStatus)
  const dispatch = useDispatch()

  const handleMenuOpen = (event) => setAnchorEl(event.currentTarget)
  const handleMenuClose = () => setAnchorEl(null)
  const triggerScan = (full) => () =>
    fetch(subsonic.url('startScan', null, { fullScan: full }))

  // Get updated status on component mount
  useEffect(() => {
    fetchUtils
      .fetchJson(subsonic.url('getScanStatus'))
      .then((resp) => resp.json['subsonic-response'])
      .then((data) => {
        if (data.status === 'ok') {
          dispatch(scanStatusUpdate(data.scanStatus))
        }
      })
  }, [dispatch])

  return (
    <div className={classes.wrapper}>
      <Tooltip title={translate('activity.title')}>
        <IconButton className={classes.button} onClick={handleMenuOpen}>
          <Badge badgeContent={null} color="secondary">
            <FiActivity size={'20'} />
          </Badge>
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
        <Card>
          <CardContent>
            <Box display="flex" className={classes.counterStatus}>
              <Box component="span" flex={2}>
                {translate('activity.totalScanned')}:
              </Box>
              <Box component="span" flex={1}>
                {scanStatus.count}
              </Box>
            </Box>
          </CardContent>
          <Divider />
          <CardActions>
            <Tooltip title={translate('activity.quickScan')}>
              <IconButton onClick={triggerScan(false)}>
                <VscSync />
              </IconButton>
            </Tooltip>
            <Tooltip title={translate('activity.fullScan')}>
              <IconButton onClick={triggerScan(true)}>
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
