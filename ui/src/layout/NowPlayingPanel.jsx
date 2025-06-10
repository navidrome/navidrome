import React, { useState, useEffect, useCallback } from 'react'
import { useSelector, useDispatch } from 'react-redux'
import { useTranslate, Link } from 'react-admin'
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
      const entryHeight = 72 // Approximate height of each ListItem
      const maxEntries = Math.min(props.entryCount || 0, 4)
      return maxEntries > 0 ? `${maxEntries * entryHeight}px` : '12em'
    },
    overflowY: 'auto',
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

const NowPlayingPanel = () => {
  const dispatch = useDispatch()
  const count = useSelector((state) => state.activity.nowPlayingCount)
  const translate = useTranslate()
  const theme = useTheme()
  const isSmallScreen = useMediaQuery(theme.breakpoints.down('sm'))

  const [anchorEl, setAnchorEl] = useState(null)
  const [entries, setEntries] = useState([])
  const open = Boolean(anchorEl)

  const classes = useStyles({ entryCount: entries.length })

  const handleMenuOpen = (event) => setAnchorEl(event.currentTarget)
  const handleMenuClose = () => setAnchorEl(null)

  // Close panel when link is clicked on small screens
  const handleLinkClick = useCallback(() => {
    if (isSmallScreen) {
      handleMenuClose()
    }
  }, [isSmallScreen])

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
          }
        })
        .catch((error) => {
          // Failed to fetch now playing data, silently ignore
        }),
    [dispatch],
  )

  // Initialize count and entries on mount
  useEffect(() => {
    fetchList()
  }, [fetchList])

  // Refresh when count changes from WebSocket events (if panel is open)
  useEffect(() => {
    if (open) fetchList()
  }, [count, open, fetchList])

  useInterval(
    () => {
      if (open) fetchList()
    },
    open ? 10000 : null,
  )

  return (
    <div>
      <Tooltip title={translate('nowPlaying.title')}>
        <IconButton className={classes.button} onClick={handleMenuOpen}>
          <Badge badgeContent={count} color="primary" overlap="rectangular" className={classes.badge}>
            <FaRegCirclePlay size={20} />
          </Badge>
        </IconButton>
      </Tooltip>
      <Popover
        id="panel-nowplaying"
        anchorEl={anchorEl}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        open={open}
        onClose={handleMenuClose}
      >
        <Card>
          <CardContent>
            {entries.length === 0 ? (
              <Typography>{translate('nowPlaying.empty')}</Typography>
            ) : (
              <List className={classes.list} dense>
                {entries.map((e) => (
                  <ListItem key={e.playerId}>
                    <ListItemAvatar>
                      <Link
                        to={`/album/${e.albumId}/show`}
                        onClick={handleLinkClick}
                      >
                        <Avatar
                          className={classes.avatar}
                          src={subsonic.getCoverArtUrl(e, 80)}
                          variant="square"
                        />
                      </Link>
                    </ListItemAvatar>
                    <ListItemText
                      primary={
                        <div className={classes.primaryText}>
                          {e.albumArtistId || e.artistId ? (
                            <Link
                              to={getArtistLink(e.albumArtistId || e.artistId)}
                              className={classes.artistLink}
                              onClick={handleLinkClick}
                            >
                              {e.albumArtist || e.artist}
                            </Link>
                          ) : (
                            <span>{e.albumArtist || e.artist}</span>
                          )}
                          &nbsp;-&nbsp;{e.title}
                        </div>
                      }
                      secondary={`${e.username}${e.playerName ? ` (${e.playerName})` : ''} • ${translate('nowPlaying.minutesAgo', { smart_count: e.minutesAgo })}`}
                    />
                  </ListItem>
                ))}
              </List>
            )}
          </CardContent>
        </Card>
      </Popover>
    </div>
  )
}

export default NowPlayingPanel
