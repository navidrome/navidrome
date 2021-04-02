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
  Avatar,
  List,
  ListItem,
  ListItemText,
  ListItemAvatar,
} from '@material-ui/core'
import { FiActivity } from 'react-icons/fi'
import { BiError } from 'react-icons/bi'
import { VscSync } from 'react-icons/vsc'
import { GiMagnifyingGlass } from 'react-icons/gi'
import subsonic from '../subsonic'
import { scanStatusUpdate } from '../actions'
import { useInterval } from '../common'
import { formatDuration } from '../utils'

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
    minWidth: '15em',
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
  const up = serverStart && serverStart.startTime
  const classes = useStyles({ up })
  const translate = useTranslate()
  const [anchorEl, setAnchorEl] = useState(null)
  const [usersCurrentlyPlaying, setUsersCurrentlyPlaying] = useState([])
  const open = Boolean(anchorEl)
  const dispatch = useDispatch()
  const scanStatus = useSelector((state) => state.activity.scanStatus)

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
    if (open) {
      fetchUtils
        .fetchJson(subsonic.url('getNowPlaying'))
        .then((resp) => resp.json['subsonic-response'])
        .then((data) => {
          if (data.status === 'ok') {
            setUsersCurrentlyPlaying(
              data.nowPlaying.entry.map((user) => {
                console.log('hey')
                return {
                  username: user.username,
                  coverArtId: user.coverArt,
                  title: user.title,
                  album: user.album,
                  artist: user.artist,
                }
              })
            )
          }
        })
    }
  }, [dispatch, anchorEl])

  return (
    <div className={classes.wrapper}>
      <Tooltip title={translate('activity.title')}>
        <IconButton className={classes.button} onClick={handleMenuOpen}>
          <Badge badgeContent={null} color="secondary">
            {up ? <FiActivity size={'20'} /> : <BiError size={'20'} />}
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
                {translate('activity.serverUptime')}:
              </Box>
              <Box component="span" flex={1}>
                {up ? <Uptime /> : translate('activity.serverDown')}
              </Box>
            </Box>
          </CardContent>
          <Divider />
          <CardContent>
            <Box display="flex" className={classes.counterStatus}>
              <Box component="span" flex={2}>
                {translate('activity.totalScanned')}:
              </Box>
              <Box component="span" flex={1}>
                {scanStatus.folderCount || '-'}
              </Box>
            </Box>
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
          <Divider />
          <List>
            {usersCurrentlyPlaying.map((user) => {
              return (
                <ListItem key={user.coverArtId} key={user.coverArtId}>
                  <ListItemAvatar>
                    <Avatar
                      alt={`${user.title} cover-art`}
                      src={subsonic.getCoverArtUrl(user, 300)}
                    />
                  </ListItemAvatar>
                  <ListItemText
                    primary={`${user.username} `}
                    secondary={`
                          ${translate('activity.currentlyPlaying')} ${
                      user.title
                    }
                          `}
                  />
                </ListItem>
              )
            })}
          </List>
        </Card>
      </Popover>
    </div>
  )
}

export default ActivityPanel
