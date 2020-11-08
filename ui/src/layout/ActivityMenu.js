import React, { useState } from 'react'
import { useSelector } from 'react-redux'
import {
  Menu,
  Badge,
  CircularProgress,
  IconButton,
  makeStyles,
  Tooltip,
  MenuItem,
} from '@material-ui/core'
import { FiActivity } from 'react-icons/fi'
import subsonic from '../subsonic'

const useStyles = makeStyles((theme) => ({
  wrapper: {
    position: 'relative',
  },
  progress: {
    position: 'absolute',
    top: -1,
    left: 0,
    zIndex: 1,
  },
  button: {
    zIndex: 2,
  },
}))

const ActivityMenu = () => {
  const classes = useStyles()
  const [anchorEl, setAnchorEl] = useState(null)
  const scanStatus = useSelector((state) => state.activity.scanStatus)

  const open = Boolean(anchorEl)

  const handleMenu = (event) => setAnchorEl(event.currentTarget)
  const handleClose = () => setAnchorEl(null)
  const startScan = () => fetch(subsonic.url('startScan', null))

  return (
    <div className={classes.wrapper}>
      <Tooltip title={'Activity'}>
        <IconButton className={classes.button} onClick={handleMenu}>
          <Badge badgeContent={null} color="secondary">
            <FiActivity size={'20'} />
          </Badge>
        </IconButton>
      </Tooltip>
      {scanStatus.scanning && (
        <CircularProgress size={46} className={classes.progress} />
      )}
      <Menu
        id="menu-activity"
        anchorEl={anchorEl}
        anchorOrigin={{
          vertical: 'top',
          horizontal: 'right',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'right',
        }}
        open={open}
        onClose={handleClose}
      >
        <MenuItem
          className={classes.root}
          activeClassName={classes.active}
          onClick={startScan}
          sidebarIsOpen={true}
        >
          {`Scanned: ${scanStatus.count}`}
        </MenuItem>
      </Menu>
    </div>
  )
}

export default ActivityMenu
