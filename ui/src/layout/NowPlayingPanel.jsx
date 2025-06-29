import React, { useState, useEffect, useCallback } from 'react'
import PropTypes from 'prop-types'
import { useSelector, useDispatch } from 'react-redux'
import { useTranslate, Link, useNotify } from 'react-admin'
import {
  Popover,
  IconButton,
  makeStyles,
  Tooltip,
  List,
  ListItem,
  ListItemText,
  ListItemAvatar,
  Avatar,
  Badge,
  Card,
  CardContent,
  Typography,
  useTheme,
  useMediaQuery,
} from '@material-ui/core'
import { FaRegCirclePlay } from 'react-icons/fa6'
import subsonic from '../subsonic'
import { useInterval } from '../common'
import { nowPlayingCountUpdate } from '../actions'
import config from '../config'

const useStyles = makeStyles((theme) => ({
  button: { color: 'inherit' },
  list: {
    width: '30em',
    maxHeight: (props) => {
      // Calculate height for up to 4 entries before scrolling
      const entryHeight = 80
      const maxEntries = Math.min(props.entryCount || 0, 4)
      return maxEntries > 0 ? `${maxEntries * entryHeight}px` : '12em'
    },
    overflowY: 'auto',
    padding: 0,
  },
  card: {
    padding: 0,
  },
  cardContent: {
    padding: `${theme.spacing(1)}px !important`, // Minimal padding, override default
    '&:last-child': {
      paddingBottom: `${theme.spacing(1)}px !important`, // Override Material-UI's last-child padding
    },
  },
  listItem: {
    paddingTop: theme.spacing(0.5),
    paddingBottom: theme.spacing(0.5),
    paddingLeft: theme.spacing(1),
    paddingRight: theme.spacing(1),
  },
  avatar: {
    width: theme.spacing(6),
    height: theme.spacing(6),
    cursor: 'pointer',
    '&:hover': {
      opacity: 0.8,
    },
  },
  badge: {
    '& .MuiBadge-badge': {
      backgroundColor: theme.palette.primary.main,
      color: theme.palette.primary.contrastText,
    },
  },
  artistLink: {
    cursor: 'pointer',
    '&:hover': {
      textDecoration: 'underline',
    },
  },
  primaryText: {
    display: 'flex',
    alignItems: 'center',
    flexWrap: 'wrap',
  },
}))

// NowPlayingButton component - handles the button with badge
const NowPlayingButton = React.memo(({ count, onClick }) => {
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <Tooltip title={translate('nowPlaying.title')}>
      <IconButton
        className={classes.button}
        onClick={onClick}
        aria-label={translate('nowPlaying.title')}
        aria-haspopup="true"
      >
        <Badge
          badgeContent={count}
          color="primary"
          overlap="rectangular"
          className={classes.badge}
        >
          <FaRegCirclePlay size={20} />
        </Badge>
      </IconButton>
    </Tooltip>
  )
})

NowPlayingButton.displayName = 'NowPlayingButton'

NowPlayingButton.propTypes = {
  count: PropTypes.number.isRequired,
  onClick: PropTypes.func.isRequired,
}

// NowPlayingItem component - individual list item
const NowPlayingItem = React.memo(
  ({ nowPlayingEntry, onLinkClick, getArtistLink }) => {
    const classes = useStyles()
    const translate = useTranslate()

    return (
      <ListItem key={nowPlayingEntry.playerId} className={classes.listItem}>
        <ListItemAvatar>
          <Link
            to={`/album/${nowPlayingEntry.albumId}/show`}
            onClick={onLinkClick}
          >
            <Avatar
              className={classes.avatar}
              src={subsonic.getCoverArtUrl(nowPlayingEntry, 80)}
              variant="square"
              alt={`${nowPlayingEntry.album} cover art`}
              loading="lazy"
            />
          </Link>
        </ListItemAvatar>
        <ListItemText
          primary={
            <div className={classes.primaryText}>
              {nowPlayingEntry.albumArtistId || nowPlayingEntry.artistId ? (
                <Link
                  to={getArtistLink(
                    nowPlayingEntry.albumArtistId || nowPlayingEntry.artistId,
                  )}
                  className={classes.artistLink}
                  onClick={onLinkClick}
                >
                  {nowPlayingEntry.albumArtist || nowPlayingEntry.artist}
                </Link>
              ) : (
                <span>
                  {nowPlayingEntry.albumArtist || nowPlayingEntry.artist}
                </span>
              )}
              &nbsp;-&nbsp;{nowPlayingEntry.title}
            </div>
          }
          secondary={`${nowPlayingEntry.username}${nowPlayingEntry.playerName ? ` (${nowPlayingEntry.playerName})` : ''} • ${translate('nowPlaying.minutesAgo', { smart_count: nowPlayingEntry.minutesAgo })}`}
        />
      </ListItem>
    )
  },
)

NowPlayingItem.displayName = 'NowPlayingItem'

NowPlayingItem.propTypes = {
  nowPlayingEntry: PropTypes.shape({
    playerId: PropTypes.oneOfType([PropTypes.string, PropTypes.number])
      .isRequired,
    albumId: PropTypes.oneOfType([PropTypes.string, PropTypes.number])
      .isRequired,
    albumArtistId: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
    artistId: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
    albumArtist: PropTypes.string,
    artist: PropTypes.string,
    title: PropTypes.string.isRequired,
    username: PropTypes.string.isRequired,
    playerName: PropTypes.string,
    minutesAgo: PropTypes.number.isRequired,
    album: PropTypes.string,
  }).isRequired,
  onLinkClick: PropTypes.func.isRequired,
  getArtistLink: PropTypes.func.isRequired,
}

// NowPlayingList component - handles the popover content
const NowPlayingList = React.memo(
  ({ anchorEl, open, onClose, entries, onLinkClick, getArtistLink }) => {
    const classes = useStyles({ entryCount: entries.length })
    const translate = useTranslate()

    return (
      <Popover
        id="panel-nowplaying"
        anchorEl={anchorEl}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        open={open}
        onClose={onClose}
        aria-labelledby="now-playing-title"
      >
        <Card className={classes.card}>
          <CardContent className={classes.cardContent}>
            {entries.length === 0 ? (
              <Typography id="now-playing-title">
                {translate('nowPlaying.empty')}
              </Typography>
            ) : (
              <List
                className={classes.list}
                dense
                aria-label={translate('nowPlaying.title')}
              >
                {entries.map((nowPlayingEntry) => (
                  <NowPlayingItem
                    key={nowPlayingEntry.playerId}
                    nowPlayingEntry={nowPlayingEntry}
                    onLinkClick={onLinkClick}
                    getArtistLink={getArtistLink}
                  />
                ))}
              </List>
            )}
          </CardContent>
        </Card>
      </Popover>
    )
  },
)

NowPlayingList.displayName = 'NowPlayingList'

NowPlayingList.propTypes = {
  anchorEl: PropTypes.object,
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
  entries: PropTypes.arrayOf(PropTypes.object).isRequired,
  onLinkClick: PropTypes.func.isRequired,
  getArtistLink: PropTypes.func.isRequired,
}

// Main NowPlayingPanel component
const NowPlayingPanel = () => {
  const dispatch = useDispatch()
  const count = useSelector((state) => state.activity.nowPlayingCount)
  const streamReconnected = useSelector(
    (state) => state.activity.streamReconnected,
  )
  const serverUp = useSelector(
    (state) => !!state.activity.serverStart.startTime,
  )
  const translate = useTranslate()
  const notify = useNotify()
  const theme = useTheme()
  const isSmallScreen = useMediaQuery(theme.breakpoints.down('sm'))

  const [anchorEl, setAnchorEl] = useState(null)
  const [entries, setEntries] = useState([])
  const open = Boolean(anchorEl)

  const handleMenuOpen = useCallback((event) => {
    setAnchorEl(event.currentTarget)
  }, [])

  const handleMenuClose = useCallback(() => {
    setAnchorEl(null)
  }, [])

  // Close panel when link is clicked on small screens
  const handleLinkClick = useCallback(() => {
    if (isSmallScreen) {
      handleMenuClose()
    }
  }, [isSmallScreen, handleMenuClose])

  const getArtistLink = useCallback((artistId) => {
    if (!artistId) return null
    return config.devShowArtistPage && artistId !== config.variousArtistsId
      ? `/artist/${artistId}/show`
      : `/album?filter={"artist_id":"${artistId}"}&order=ASC&sort=max_year&displayedFilters={"compilation":true}&perPage=15`
  }, [])

  const fetchList = useCallback(
    () =>
      subsonic
        .getNowPlaying()
        .then((resp) => resp.json['subsonic-response'])
        .then((data) => {
          if (data.status === 'ok') {
            const nowPlayingEntries = data.nowPlaying?.entry || []
            setEntries(nowPlayingEntries)
            // Also update the count in Redux store
            dispatch(nowPlayingCountUpdate({ count: nowPlayingEntries.length }))
          } else {
            throw new Error(
              data.error?.message || 'Failed to fetch now playing data',
            )
          }
        })
        .catch((error) => {
          notify('ra.page.error', 'warning', {
            messageArgs: { error: error.message || 'Unknown error' },
          })
        }),
    [dispatch, notify],
  )

  // Initialize count and entries on mount, and refresh on server/stream changes
  useEffect(() => {
    if (serverUp) fetchList()
  }, [fetchList, serverUp, streamReconnected])

  // Refresh when count changes from WebSocket events (if panel is open)
  useEffect(() => {
    if (open && serverUp) fetchList()
  }, [count, open, fetchList, serverUp])

  // Periodic refresh when panel is open (10 seconds)
  useInterval(
    () => {
      if (open && serverUp) fetchList()
    },
    open ? 10000 : null,
  )

  // Periodic refresh when panel is closed (60 seconds) to keep badge accurate
  useInterval(
    () => {
      if (!open && serverUp) fetchList()
    },
    !open ? 60000 : null,
  )

  return (
    <div>
      <NowPlayingButton count={count} onClick={handleMenuOpen} />
      <NowPlayingList
        anchorEl={anchorEl}
        open={open}
        onClose={handleMenuClose}
        entries={entries}
        onLinkClick={handleLinkClick}
        getArtistLink={getArtistLink}
      />
    </div>
  )
}

NowPlayingPanel.propTypes = {}

export default NowPlayingPanel
