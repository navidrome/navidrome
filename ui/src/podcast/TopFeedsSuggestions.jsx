import { useEffect, useState } from 'react'
import { useDataProvider, useNotify, useTranslate } from 'react-admin'
import {
  Avatar,
  Box,
  Chip,
  CircularProgress,
  MenuItem,
  Select,
  Typography,
  makeStyles,
} from '@material-ui/core'
import MicIcon from '@material-ui/icons/Mic'
import { REST_URL } from '../consts'
import { httpClient } from '../dataProvider'
import detectCountry from './detectCountry'
import countryOptions from './countryOptions'

const STORAGE_KEY = 'nd_podcast_top_country'

const useStyles = makeStyles({
  header: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    marginBottom: '0.5rem',
  },
  chips: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: '0.5rem',
  },
})

const getStoredCountry = () => {
  try {
    return localStorage.getItem(STORAGE_KEY) || detectCountry()
  } catch {
    return detectCountry()
  }
}

// Shows the current top-podcasts chart for a region (auto-detected via
// timezone, overridable and remembered per-browser) as one-click "quick
// add" suggestions. Used both on the empty-state Podcasts list and on the
// Add Podcast page, so it's always reachable - not just on first run.
const TopFeedsSuggestions = ({ onSubscribed, title }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const [country, setCountry] = useState(getStoredCountry)
  const [feeds, setFeeds] = useState(null)
  const [adding, setAdding] = useState(null)

  useEffect(() => {
    setFeeds(null)
    httpClient(`${REST_URL}/podcastChannel/top?country=${country}`)
      .then(({ json }) => setFeeds(json || []))
      .catch(() => setFeeds([]))
  }, [country])

  const handleCountryChange = (e) => {
    const value = e.target.value
    setCountry(value)
    try {
      localStorage.setItem(STORAGE_KEY, value)
    } catch {
      // ignore storage failures (e.g. private browsing)
    }
  }

  const handleAdd = (feed) => {
    setAdding(feed.feedUrl)
    dataProvider
      .create('podcastChannel', { data: { url: feed.feedUrl } })
      .then(() => {
        notify('resources.podcastChannel.notifications.subscribed', {
          type: 'info',
        })
        onSubscribed && onSubscribed()
      })
      .catch(() => {
        notify('resources.podcastChannel.notifications.subscribeFailed', {
          type: 'warning',
        })
      })
      .finally(() => setAdding(null))
  }

  return (
    <Box>
      <Box className={classes.header}>
        {title && (
          <Typography variant="body2" color="textSecondary">
            {title}
          </Typography>
        )}
        <Select
          value={country}
          onChange={handleCountryChange}
          variant="outlined"
          margin="dense"
        >
          {countryOptions.map((c) => (
            <MenuItem key={c.code} value={c.code}>
              {c.name}
            </MenuItem>
          ))}
        </Select>
      </Box>

      {feeds === null && <CircularProgress size={20} />}

      {feeds && feeds.length === 0 && (
        <Typography variant="body2" color="textSecondary">
          {translate('resources.podcastChannel.noSearchResults')}
        </Typography>
      )}

      {feeds && feeds.length > 0 && (
        <Box className={classes.chips}>
          {feeds.map((feed) => (
            <Chip
              key={feed.feedUrl}
              avatar={
                <Avatar src={feed.artworkUrl}>
                  <MicIcon />
                </Avatar>
              }
              label={feed.title}
              clickable
              disabled={!!adding}
              onClick={() => handleAdd(feed)}
            />
          ))}
        </Box>
      )}
    </Box>
  )
}

export default TopFeedsSuggestions
