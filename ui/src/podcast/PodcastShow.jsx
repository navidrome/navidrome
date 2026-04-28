import React, { useEffect, useState } from 'react'
import {
  Card,
  CardContent,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  makeStyles,
  Link,
} from '@material-ui/core'
import { Button, useTranslate, useShowController, Title } from 'react-admin'
import { useDispatch } from 'react-redux'
import MicIcon from '@material-ui/icons/Mic'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import { RiPlayList2Fill, RiPlayListAddFill } from 'react-icons/ri'
import StatusBadge from './StatusBadge'
import EpisodeActions from './EpisodeActions'
import subsonic from '../subsonic'
import { setTrack, playTracks, shuffleTracks, playNext, addTracks } from '../actions'

const songFromEpisode = (episode, channelTitle) => ({
  id: episode.streamId,
  title: episode.title,
  album: channelTitle || episode.channelId,
  artist: '',
  duration: episode.duration,
  suffix: episode.suffix,
  isPodcast: true,
  channelId: episode.channelId,
})

const buildTracksData = (episodes, channelTitle) => {
  const data = {}
  const ids = []
  episodes
    .filter((ep) => ep.status === 'completed' && ep.streamId)
    .forEach((ep) => {
      const song = songFromEpisode(ep, channelTitle)
      data[song.id] = song
      ids.push(song.id)
    })
  return { data, ids }
}

const EpisodePlayButtons = ({ episodes, channelTitle }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const { data, ids } = buildTracksData(episodes, channelTitle)
  if (!ids.length) return null
  return (
    <div style={{ marginBottom: 8 }}>
      <Button onClick={() => dispatch(playTracks(data, ids))} label={translate('resources.album.actions.playAll')}>
        <PlayArrowIcon />
      </Button>
      <Button onClick={() => dispatch(shuffleTracks(data, ids))} label={translate('resources.album.actions.shuffle')}>
        <ShuffleIcon />
      </Button>
      <Button onClick={() => dispatch(playNext(data, ids))} label={translate('resources.album.actions.playNext')}>
        <RiPlayList2Fill />
      </Button>
      <Button onClick={() => dispatch(addTracks(data, ids))} label={translate('resources.album.actions.addToQueue')}>
        <RiPlayListAddFill />
      </Button>
    </div>
  )
}

const useStyles = makeStyles((theme) => ({
  card: { marginTop: theme.spacing(2) },
  header: { display: 'flex', gap: theme.spacing(2), marginBottom: theme.spacing(3) },
  playableRow: { cursor: 'pointer', '&:hover': { backgroundColor: theme.palette.action.hover } },
  avatar: {
    width: 192,
    height: 192,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.palette.grey[300],
    borderRadius: 4,
    flexShrink: 0,
  },
  meta: { flex: 1 },
  description: { marginTop: theme.spacing(1), color: theme.palette.text.secondary },
  tableWrapper: { marginTop: theme.spacing(2), overflowX: 'auto' },
}))

const buildSrcSet = (images) => {
  if (!images || images.length === 0) return undefined
  return images.map((img) => `${img.url} ${img.width}w`).join(', ')
}

const formatDuration = (seconds) => {
  if (!seconds) return ''
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}


const PodcastShow = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const dispatch = useDispatch()
  const { record } = useShowController(props)
  const [episodes, setEpisodes] = useState([])

  const loadEpisodes = () => {
    if (!record?.id) return
    subsonic
      .getPodcasts(record.id, true)
      .then((res) => {
        const channels = res?.json?.['subsonic-response']?.podcasts?.channel || []
        const ch = channels.find((c) => c.id === record.id)
        setEpisodes(ch?.episode || [])
      })
      .catch(() => {})
  }

  useEffect(loadEpisodes, [record?.id])

  // Subscribe to SSE progress only while episodes are downloading
  const hasDownloading = episodes.some((ep) => ep.status === 'downloading')
  useEffect(() => {
    if (!hasDownloading) return
    const handler = (e) => {
      const { episodeId, downloadedBytes, size, duration, status } = e.detail
      if (status === 'completed' || status === 'error') {
        // Reload to get updated streamId and full episode data
        loadEpisodes()
        return
      }
      setEpisodes((prev) =>
        prev.map((ep) =>
          ep.id === episodeId
            ? { ...ep, downloadedBytes, size, ...(duration ? { duration } : {}) }
            : ep,
        ),
      )
    }
    window.addEventListener('podcastEpisodeProgress', handler)
    return () => window.removeEventListener('podcastEpisodeProgress', handler)
  }, [hasDownloading])

  if (!record) return null

  return (
    <Card className={classes.card}>
      <Title subTitle={record.title} />
      <CardContent>
        <div className={classes.header}>
          <div className={classes.avatar}>
            {record.imageUrl ? (
              <img
                src={record.imageUrl}
                srcSet={buildSrcSet(record.images)}
                sizes="192px"
                alt={record.title}
                style={{ width: '100%', height: '100%', objectFit: 'cover', borderRadius: 4 }}
              />
            ) : (
              <MicIcon style={{ fontSize: 40, color: '#888' }} />
            )}
          </div>
          <div className={classes.meta}>
            <Typography variant="h5">{record.title}</Typography>
            <Link href={record.url} target="_blank" rel="noopener noreferrer" variant="body2">
              {record.url}
            </Link>
            {record.description && (
              <Typography variant="body2" className={classes.description}>
                {record.description}
              </Typography>
            )}
            {record.status === 'error' && (
              <StatusBadge status="error" errorMessage={record.errorMessage} />
            )}
          </div>
        </div>

        <EpisodePlayButtons episodes={episodes} channelTitle={record.title} />

        <div className={classes.tableWrapper}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>{translate('resources.podcast.fields.title')}</TableCell>
                <TableCell>{translate('resources.podcast.fields.publishDate')}</TableCell>
                <TableCell>{translate('resources.podcast.fields.duration')}</TableCell>
                <TableCell>{translate('resources.podcast.fields.status')}</TableCell>
                <TableCell />
              </TableRow>
            </TableHead>
            <TableBody>
              {episodes.map((ep) => (
                <TableRow
                  key={ep.id}
                  className={ep.status === 'completed' ? classes.playableRow : undefined}
                  onClick={() => ep.status === 'completed' && dispatch(setTrack(songFromEpisode(ep, record.title)))}
                >
                  <TableCell>{ep.title}</TableCell>
                  <TableCell>
                    {ep.publishDate ? new Date(ep.publishDate).toLocaleDateString() : ''}
                  </TableCell>
                  <TableCell>{formatDuration(ep.duration)}</TableCell>
                  <TableCell>
                    <StatusBadge status={ep.status} errorMessage={ep.errorMessage} downloadedBytes={ep.downloadedBytes} size={ep.size} />
                  </TableCell>
                  <TableCell>
                    <EpisodeActions episode={ep} onRefresh={loadEpisodes} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}

export default PodcastShow
