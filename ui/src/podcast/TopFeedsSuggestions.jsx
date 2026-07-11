import { useEffect, useState } from 'react'
import { useDataProvider, useNotify } from 'react-admin'
import {
  Avatar,
  Box,
  Chip,
  CircularProgress,
  Typography,
  makeStyles,
} from '@material-ui/core'
import MicIcon from '@material-ui/icons/Mic'
import { REST_URL } from '../consts'
import { httpClient } from '../dataProvider'
import detectCountry from './detectCountry'

const useStyles = makeStyles({
  chips: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: '0.5rem',
  },
})

// Shows the current top-podcasts chart for the user's likely region as
// one-click "quick add" suggestions. Used both on the empty-state Podcasts
// list and on the Add Podcast page, so it's always reachable - not just on
// first run.
const TopFeedsSuggestions = ({ onSubscribed, title }) => {
  const classes = useStyles()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const [feeds, setFeeds] = useState(null)
  const [adding, setAdding] = useState(null)

  useEffect(() => {
    const country = detectCountry()
    httpClient(`${REST_URL}/podcastChannel/top?country=${country}`)
      .then(({ json }) => setFeeds(json || []))
      .catch(() => setFeeds([]))
  }, [])

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

  if (feeds === null) {
    return <CircularProgress size={20} />
  }
  if (feeds.length === 0) {
    return null
  }

  return (
    <Box>
      {title && (
        <Typography variant="body2" color="textSecondary" gutterBottom>
          {title}
        </Typography>
      )}
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
    </Box>
  )
}

export default TopFeedsSuggestions
