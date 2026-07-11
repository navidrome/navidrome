import React from 'react'
import {
  ReferenceManyField,
  ShowContextProvider,
  useShowContext,
  useShowController,
  Title as RaTitle,
  Datagrid,
  TextField,
  DateField,
  FunctionField,
  SimpleShowLayout,
  useTranslate,
  useNotify,
  Pagination,
} from 'react-admin'
import { useDispatch } from 'react-redux'
import {
  Typography,
  Box,
  Avatar,
  Chip,
  IconButton,
  makeStyles,
} from '@material-ui/core'
import MicIcon from '@material-ui/icons/Mic'
import DownloadIcon from '@material-ui/icons/GetApp'
import DeleteIcon from '@material-ui/icons/Delete'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import { Title, useResourceRefresh } from '../common'
import { setTrack, openAddToPlaylist } from '../actions'
import subsonic from '../subsonic'
import config from '../config'
import { REST_URL } from '../consts'
import { httpClient } from '../dataProvider'
import { songFromPodcastEpisode } from './helper'

const useStyles = makeStyles({
  header: {
    display: 'flex',
    alignItems: 'center',
    gap: '1rem',
    marginBottom: '1rem',
  },
  cover: {
    width: '5rem',
    height: '5rem',
  },
})

const statusColor = {
  downloaded: 'primary',
  downloading: 'default',
  queued: 'default',
  error: 'secondary',
}

const PodcastChannelHeader = () => {
  const { record } = useShowContext()
  const classes = useStyles()
  if (!record) return null
  const cover =
    record.uploadedImage || record.coverArtUrl
      ? subsonic.getCoverArtUrl(record, config.uiCoverArtSize, true)
      : undefined
  return (
    <Box className={classes.header}>
      <Avatar src={cover} variant="rounded" className={classes.cover}>
        <MicIcon />
      </Avatar>
      <Box>
        <Typography variant="h6">{record.title}</Typography>
        {record.description && (
          <Typography variant="body2" color="textSecondary">
            {record.description}
          </Typography>
        )}
      </Box>
    </Box>
  )
}

const DownloadStatusChip = ({ record }) => {
  const translate = useTranslate()
  if (!record) return null
  return (
    <Chip
      size="small"
      label={translate(
        `resources.podcastEpisode.downloadStatus.${record.downloadStatus}`,
        { _: record.downloadStatus },
      )}
      color={statusColor[record.downloadStatus] || 'default'}
      variant={record.downloadStatus === 'downloaded' ? 'default' : 'outlined'}
    />
  )
}

const EpisodeActions = ({ record, isAdmin }) => {
  const dispatch = useDispatch()
  const notify = useNotify()
  if (!record) return null

  const stop = (e) => e.stopPropagation()

  const handleDownload = (e) => {
    stop(e)
    httpClient(`${REST_URL}/podcastEpisode/${record.id}/download`, {
      method: 'POST',
    }).catch(() => notify('ra.page.error', { type: 'warning' }))
  }

  const handleDelete = (e) => {
    stop(e)
    httpClient(`${REST_URL}/podcastEpisode/${record.id}`, {
      method: 'DELETE',
    }).catch(() => notify('ra.page.error', { type: 'warning' }))
  }

  const handleAddToPlaylist = (e) => {
    stop(e)
    dispatch(openAddToPlaylist({ selectedIds: [record.id] }))
  }

  // Only downloaded episodes can be added to a playlist - a playlist entry
  // has no way to represent "stream this from the source URL".
  const isDownloaded = record.downloadStatus === 'downloaded'

  return (
    <>
      {isDownloaded && (
        <IconButton size="small" onClick={handleAddToPlaylist} onFocus={stop}>
          <PlaylistAddIcon fontSize="small" />
        </IconButton>
      )}
      {isAdmin &&
        (isDownloaded ||
        record.downloadStatus === 'downloading' ||
        record.downloadStatus === 'queued' ? (
          <IconButton size="small" onClick={handleDelete} onFocus={stop}>
            <DeleteIcon fontSize="small" />
          </IconButton>
        ) : (
          <IconButton size="small" onClick={handleDownload} onFocus={stop}>
            <DownloadIcon fontSize="small" />
          </IconButton>
        ))}
    </>
  )
}

const EpisodesSection = ({ channel, isAdmin }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()

  const handleRowClick = (id, basePath, record) => {
    dispatch(setTrack(songFromPodcastEpisode(record, channel)))
    return false
  }

  return (
    <>
      <Box mt={2} mb={1}>
        <Typography variant="h6">
          {translate('resources.podcastEpisode.name', { smart_count: 2 })}
        </Typography>
      </Box>
      <Datagrid rowClick={handleRowClick}>
        <TextField source="title" />
        <DateField source="publishDate" />
        <FunctionField
          source="duration"
          render={(record) => {
            if (!record.duration) return null
            const mins = Math.floor(record.duration / 60)
            const secs = Math.floor(record.duration % 60)
              .toString()
              .padStart(2, '0')
            return `${mins}:${secs}`
          }}
        />
        <FunctionField
          source="downloadStatus"
          render={(record) => <DownloadStatusChip record={record} />}
        />
        <FunctionField
          source="id"
          label=""
          sortable={false}
          render={(record) => (
            <EpisodeActions record={record} isAdmin={isAdmin} />
          )}
        />
      </Datagrid>
    </>
  )
}

const PodcastChannelShowLayout = ({ permissions, ...props }) => {
  const { record, loading } = useShowContext(props)
  useResourceRefresh('podcastChannel', 'podcastEpisode')
  const isAdmin = permissions === 'admin'

  if (loading || !record) return null

  return (
    <>
      <RaTitle title={<Title subTitle={record.title} />} />
      <SimpleShowLayout>
        <PodcastChannelHeader />
        <ReferenceManyField
          reference="podcastEpisode"
          target="channel_id"
          label=""
          sort={{ field: 'publishDate', order: 'DESC' }}
          perPage={100}
          pagination={<Pagination rowsPerPageOptions={[50, 100, 250]} />}
          fullWidth
        >
          <EpisodesSection channel={record} isAdmin={isAdmin} />
        </ReferenceManyField>
      </SimpleShowLayout>
    </>
  )
}

const PodcastChannelShow = (props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <PodcastChannelShowLayout {...props} {...controllerProps} />
    </ShowContextProvider>
  )
}

export default PodcastChannelShow
