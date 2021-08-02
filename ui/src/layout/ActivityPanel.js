import React, { useState, useEffect } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useNotify, useTranslate } from 'react-admin'
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
  Typography,
  List,
  ListItem,
  ListItemText,
  ListItemAvatar,
  ListItemSecondaryAction,
} from '@material-ui/core'
import { FiActivity } from 'react-icons/fi'
import { BiError } from 'react-icons/bi'
import { VscSync } from 'react-icons/vsc'
import { GiMagnifyingGlass } from 'react-icons/gi'
import subsonic from '../subsonic'
import { scanStatusUpdate } from '../actions'
import { useInterval } from '../common'
import { formatDuration } from '../utils'
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
    minWidth: '15em',
  },
  heading: {
    padding: '0px 16px',
  },
  list: {
    maxWidth: '300px',
    textOverflow: 'ellipsis',
    ' & .primary': {
      '& .MuiListItemText-primary': {
        marginRight: '80px',
        wordWrap: 'break-word',
      },
      '& .MuiListItemText-secondary': {
        whiteSpace: 'nowrap',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
      },
    },

    '& .MuiListItemSecondaryAction-root': {
      '& .MuiListItemText-primary': {
        textAlign: 'right',
        fontSize: '12px',
      },
      '& .MuiListItemText-secondary': {
        visibility: 'hidden',
      },
    },
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
  const classes = useStyles({ up })
  const translate = useTranslate()
  const notify = useNotify()
  const [anchorEl, setAnchorEl] = useState(null)
  const [nowPlaying, setNowPlaying] = useState([])
  const open = Boolean(anchorEl)
  const dispatch = useDispatch()
  const scanStatus = useSelector((state) => state.activity.scanStatus)

  const handleMenuOpen = (event) => setAnchorEl(event.currentTarget)
  const handleMenuClose = () => setAnchorEl(null)
  const triggerScan = (full) => () => subsonic.startScan({ fullScan: full })

  // Get updated status on component mount
  useEffect(() => {
    subsonic
      .getScanStatus()
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
            setNowPlaying(
              data?.nowPlaying?.entry?.map((user) => {
                return {
                  username: user.username,
                  coverArtId: user.coverArt,
                  title: user.title,
                  album: user.album,
                  artist: user.artist,
                  minutesAgo: user.minutesAgo || 0,
                }
              }) || []
            )
          }
        })
    }
  }, [dispatch, open])

  useEffect(() => {
    if (serverStart.version && serverStart.version !== config.version) {
      notify('ra.notification.new_version', 'info', {}, false, 604800000 * 50)
    }
  }, [serverStart, notify])

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
          <Typography className={classes.heading}>
            {translate('activity.nowPlaying')}
          </Typography>
          <List className={classes.list}>
            {nowPlaying.map((user) => {
              return (
                <ListItem key={user.coverArtId}>
                  <ListItemAvatar>
                    <Avatar
                      alt={`${user.title} ${translate('activity.coverArt')}`}
                      src={subsonic.getCoverArtUrl(user)}
                    />
                  </ListItemAvatar>
                  <ListItemText
                    className="primary"
                    primary={`${user.username}`}
                    secondary={`${user.title} - ${user.artist}`}
                  />
                  <ListItemSecondaryAction>
                    <ListItemText
                      primary={`${user.minutesAgo} ${translate(
                        'activity.minutesAgo'
                      )}`}
                      secondary="place-holder"
                    ></ListItemText>
                  </ListItemSecondaryAction>
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
