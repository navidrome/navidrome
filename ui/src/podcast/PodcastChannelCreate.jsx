import { useState } from 'react'
import {
  Create,
  required,
  SimpleForm,
  TextInput,
  useDataProvider,
  useNotify,
  useRedirect,
  useTranslate,
} from 'react-admin'
import {
  Avatar,
  Box,
  Button,
  CircularProgress,
  List,
  ListItem,
  ListItemAvatar,
  ListItemSecondaryAction,
  ListItemText,
  TextField as MuiTextField,
  Typography,
  makeStyles,
} from '@material-ui/core'
import SearchIcon from '@material-ui/icons/Search'
import MicIcon from '@material-ui/icons/Mic'
import { Title } from '../common'
import { REST_URL } from '../consts'
import { httpClient } from '../dataProvider'
import { urlValidate } from '../utils/validations'

const useStyles = makeStyles({
  searchRow: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    marginBottom: '1rem',
  },
  avatar: {
    width: 48,
    height: 48,
  },
  manualEntry: {
    marginTop: '1rem',
  },
})

const PodcastChannelTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.podcastChannel.name', {
    smart_count: 1,
  })
  const title = translate('ra.page.create', { name: `${resourceName}` })
  return <Title subTitle={title} />
}

const PodcastSearch = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const redirect = useRedirect()
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [subscribing, setSubscribing] = useState(null)
  const [results, setResults] = useState(null)

  const handleSearch = (e) => {
    e.preventDefault()
    if (!query.trim()) return
    setLoading(true)
    httpClient(
      `${REST_URL}/podcastChannel/search?q=${encodeURIComponent(query)}`,
    )
      .then(({ json }) => setResults(json || []))
      .catch(() => {
        notify('resources.podcastChannel.notifications.searchFailed', {
          type: 'warning',
        })
        setResults([])
      })
      .finally(() => setLoading(false))
  }

  const handleSubscribe = (feedUrl) => {
    setSubscribing(feedUrl)
    dataProvider
      .create('podcastChannel', { data: { url: feedUrl } })
      .then(() => {
        redirect('/podcastChannel')
      })
      .catch(() => {
        notify('resources.podcastChannel.notifications.subscribeFailed', {
          type: 'warning',
        })
        setSubscribing(null)
      })
  }

  return (
    <Box p={2}>
      <form onSubmit={handleSearch} className={classes.searchRow}>
        <MuiTextField
          variant="outlined"
          size="small"
          fullWidth
          placeholder={translate('resources.podcastChannel.searchPlaceholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Button
          type="submit"
          variant="outlined"
          startIcon={<SearchIcon />}
          disabled={loading}
        >
          {translate('resources.podcastChannel.search')}
        </Button>
      </form>

      {loading && <CircularProgress size={24} />}

      {results && results.length === 0 && !loading && (
        <Typography variant="body2" color="textSecondary">
          {translate('resources.podcastChannel.noSearchResults')}
        </Typography>
      )}

      {results && results.length > 0 && (
        <List>
          {results.map((r) => (
            <ListItem key={r.feedUrl}>
              <ListItemAvatar>
                <Avatar src={r.artworkUrl} className={classes.avatar}>
                  <MicIcon />
                </Avatar>
              </ListItemAvatar>
              <ListItemText primary={r.title} secondary={r.author} />
              <ListItemSecondaryAction>
                <Button
                  variant="outlined"
                  size="small"
                  disabled={!!subscribing}
                  onClick={() => handleSubscribe(r.feedUrl)}
                >
                  {subscribing === r.feedUrl
                    ? translate('resources.podcastChannel.subscribing')
                    : translate('resources.podcastChannel.subscribe')}
                </Button>
              </ListItemSecondaryAction>
            </ListItem>
          ))}
        </List>
      )}
    </Box>
  )
}

const PodcastChannelCreate = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  return (
    <>
      <PodcastSearch />
      <Box className={classes.manualEntry}>
        <Create title={<PodcastChannelTitle />} {...props}>
          <SimpleForm redirect="list" variant={'outlined'}>
            <Typography variant="body2" color="textSecondary" gutterBottom>
              {translate('resources.podcastChannel.manualEntryLabel')}
            </Typography>
            <TextInput
              type="url"
              source="url"
              fullWidth
              validate={[required(), urlValidate]}
            />
          </SimpleForm>
        </Create>
      </Box>
    </>
  )
}

export default PodcastChannelCreate
