import React, { useState, useEffect } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { fetchUtils } from 'react-admin'
import {
  Menu,
  MenuItem,
  Badge,
  CircularProgress,
  IconButton,
  makeStyles,
  Tooltip,
} from '@material-ui/core'
import { FiActivity } from 'react-icons/fi'
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
}))

const ActivityMenu = () => {
  const classes = useStyles()
  const [anchorEl, setAnchorEl] = useState(null)
  const open = Boolean(anchorEl)
  const scanStatus = useSelector((state) => state.activity.scanStatus)
  const dispatch = useDispatch()

  const handleMenuOpen = (event) => setAnchorEl(event.currentTarget)
  const handleCloseClose = () => setAnchorEl(null)
  const triggerScan = () => fetch(subsonic.url('startScan'))

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
      <Tooltip title={'Activity'}>
        <IconButton className={classes.button} onClick={handleMenuOpen}>
          <Badge badgeContent={null} color="secondary">
            <FiActivity size={'20'} />
          </Badge>
        </IconButton>
      </Tooltip>
      {scanStatus.scanning && (
        <CircularProgress size={24} className={classes.progress} />
      )}
      <Menu
        id="menu-activity"
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
        onClose={handleCloseClose}
      >
        <MenuItem
          className={classes.root}
          activeClassName={classes.active}
          onClick={triggerScan}
          sidebarIsOpen={true}
        >
          {`Scanned: ${scanStatus.count}`}
        </MenuItem>
      </Menu>
    </div>
  )
}

export default ActivityMenu
