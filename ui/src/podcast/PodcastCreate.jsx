import React, { useState } from 'react'
import { useTranslate, useNotify, useRedirect, useRefresh, Title } from 'react-admin'
import {
  Avatar,
  Card,
  CardContent,
  CircularProgress,
  Divider,
  InputAdornment,
  TextField,
  Typography,
  makeStyles,
} from '@material-ui/core'
import { Button } from 'react-admin'
import MicIcon from '@material-ui/icons/Mic'
import SearchIcon from '@material-ui/icons/Search'
import AddIcon from '@material-ui/icons/Add'
import subsonic from '../subsonic'

const useStyles = makeStyles((theme) => ({
  root: { marginTop: theme.spacing(2) },
  urlRow: { display: 'flex', gap: theme.spacing(1), alignItems: 'flex-start' },
  urlInput: { flex: 1 },
  preview: {
    marginTop: theme.spacing(3),
    display: 'flex',
    gap: theme.spacing(2),
    alignItems: 'flex-start',
  },
  previewImage: { width: 120, height: 120, borderRadius: 4, flexShrink: 0 },
  previewInfo: { flex: 1 },
  previewTitle: { fontWeight: 600, marginBottom: theme.spacing(0.5) },
  previewDesc: { color: theme.palette.text.secondary, marginBottom: theme.spacing(1) },
  addButton: { marginTop: theme.spacing(2) },
}))

const PodcastCreate = () => {
  const translate = useTranslate()
  const notify = useNotify()
  const redirect = useRedirect()
  const refresh = useRefresh()
  const classes = useStyles()
  const [feedUrl, setFeedUrl] = useState('')
  const [fetching, setFetching] = useState(false)
  const [adding, setAdding] = useState(false)
  const [preview, setPreview] = useState(null)

  const title = translate('ra.page.create', {
    name: translate('resources.podcast.name', { smart_count: 1 }),
  })

  const handleFetch = async () => {
    if (!feedUrl) return
    setFetching(true)
    setPreview(null)
    try {
      const res = await subsonic.previewPodcastFeed(feedUrl)
      setPreview(res.json)
    } catch {
      notify('ra.notification.http_error', { type: 'error' })
    } finally {
      setFetching(false)
    }
  }

  const handleAdd = async () => {
    setAdding(true)
    try {
      await subsonic.createPodcastChannel(feedUrl)
      notify('resources.podcast.notifications.channelAdded')
      redirect('/podcast')
      refresh()
    } catch {
      notify('ra.notification.http_error', { type: 'error' })
    } finally {
      setAdding(false)
    }
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      handleFetch()
    }
  }

  return (
    <Card className={classes.root}>
      <Title subTitle={title} />
      <CardContent>
        <div className={classes.urlRow}>
          <TextField
            className={classes.urlInput}
            label={translate('resources.podcast.fields.url')}
            value={feedUrl}
            onChange={(e) => { setFeedUrl(e.target.value); setPreview(null) }}
            onKeyDown={handleKeyDown}
            type="url"
            variant="outlined"
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <MicIcon color="action" />
                </InputAdornment>
              ),
            }}
          />
          <Button
            variant="contained"
            color="primary"
            onClick={handleFetch}
            disabled={fetching || !feedUrl}
            label="resources.podcast.actions.fetchFeed"
            style={{ marginTop: 8 }}
          >
            {fetching ? <CircularProgress size={18} color="inherit" /> : <SearchIcon />}
          </Button>
        </div>

        {preview && (
          <>
            <Divider style={{ marginTop: 24, marginBottom: 8 }} />
            <div className={classes.preview}>
              {preview.imageUrl ? (
                <img src={preview.imageUrl} alt={preview.title} className={classes.previewImage} />
              ) : (
                <Avatar variant="rounded" className={classes.previewImage}>
                  <MicIcon style={{ fontSize: 48 }} />
                </Avatar>
              )}
              <div className={classes.previewInfo}>
                <Typography variant="h6" className={classes.previewTitle}>
                  {preview.title}
                </Typography>
                {preview.episodeCount > 0 && (
                  <Typography variant="body2" color="textSecondary">
                    {translate('resources.podcast.fields.episodeCount')}: {preview.episodeCount}
                  </Typography>
                )}
                {preview.description && (
                  <Typography variant="body2" className={classes.previewDesc}>
                    {preview.description}
                  </Typography>
                )}
                {preview.alreadyExists ? (
                  <Typography variant="body2" color="error" style={{ marginTop: 8 }}>
                    {translate('resources.podcast.notifications.alreadyExists')}
                  </Typography>
                ) : (
                  <Button
                    className={classes.addButton}
                    variant="contained"
                    color="primary"
                    onClick={handleAdd}
                    disabled={adding}
                    label="resources.podcast.actions.addChannel"
                  >
                    {adding ? <CircularProgress size={18} color="inherit" /> : <AddIcon />}
                  </Button>
                )}
              </div>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}

export default PodcastCreate
