import React, { useState } from 'react'
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
  useRefresh,
  Pagination,
} from 'react-admin'
import { useDispatch } from 'react-redux'
import {
  Typography,
  Box,
  Avatar,
  Button,
  Chip,
  CircularProgress,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  TextField as MuiTextField,
  Tooltip,
  makeStyles,
} from '@material-ui/core'
import MicIcon from '@material-ui/icons/Mic'
import DownloadIcon from '@material-ui/icons/GetApp'
import DeleteIcon from '@material-ui/icons/Delete'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import CheckCircleIcon from '@material-ui/icons/CheckCircle'
import CheckCircleOutlineIcon from '@material-ui/icons/CheckCircleOutline'
import RefreshIcon from '@material-ui/icons/Refresh'
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
  headerText: {
    flexGrow: 1,
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

// refreshNow triggers RefreshChannel synchronously on the server (real feed fetch, not just a UI
// reload) - available to any subscriber, not just admins, matching the backend endpoint's own
// permission model (any subscriber can trigger a refresh of a shared channel).
const PodcastChannelHeader = () => {
  const { record } = useShowContext()
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const refreshList = useRefresh()
  const [refreshing, setRefreshing] = useState(false)
  if (!record) return null
  const cover =
    record.uploadedImage || record.coverArtUrl
      ? subsonic.getCoverArtUrl(record, config.uiCoverArtSize, true)
      : undefined

  const handleRefresh = () => {
    setRefreshing(true)
    httpClient(`${REST_URL}/podcastChannel/${record.id}/refresh`, {
      method: 'POST',
    })
      .then(() => {
        notify('resources.podcastChannel.notifications.refreshed', {
          type: 'info',
        })
        refreshList()
      })
      .catch(() =>
        notify('resources.podcastChannel.notifications.refreshFailed', {
          type: 'warning',
        }),
      )
      .finally(() => setRefreshing(false))
  }

  return (
    <Box className={classes.header}>
      <Avatar src={cover} variant="rounded" className={classes.cover}>
        <MicIcon />
      </Avatar>
      <Box className={classes.headerText}>
        <Typography variant="h6">{record.title}</Typography>
        {record.description && (
          <Typography variant="body2" color="textSecondary">
            {record.description}
          </Typography>
        )}
      </Box>
      <Button
        variant="outlined"
        onClick={handleRefresh}
        disabled={refreshing}
        startIcon={
          refreshing ? <CircularProgress size={16} /> : <RefreshIcon />
        }
      >
        {translate(
          refreshing
            ? 'resources.podcastChannel.refreshing'
            : 'resources.podcastChannel.refreshNow',
        )}
      </Button>
    </Box>
  )
}

const downloadPolicyChoices = ['none', 'new', 'all']

// Lets the current user manage their own download policy/retention settings for this channel -
// available to any subscriber (not just admins), since download policy/retention are per-user
// now (see model.PodcastSubscription). Not present when record.subscription is nil (e.g. an
// admin viewing a channel they haven't personally subscribed to).
const SubscriptionSettings = ({ record }) => {
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const sub = record?.subscription
  const [downloadPolicy, setDownloadPolicy] = useState(sub?.downloadPolicy)
  const [retentionCount, setRetentionCount] = useState(sub?.retentionCount)
  const [retentionDays, setRetentionDays] = useState(sub?.retentionDays)
  const [maxStorageMb, setMaxStorageMb] = useState(sub?.maxStorageMb)
  const [saving, setSaving] = useState(false)

  if (!sub) return null

  const handleSave = () => {
    setSaving(true)
    httpClient(`${REST_URL}/podcastChannel/${record.id}/subscription`, {
      method: 'PUT',
      body: JSON.stringify({
        downloadPolicy,
        retentionCount: Number(retentionCount) || 0,
        retentionDays: Number(retentionDays) || 0,
        maxStorageMb: Number(maxStorageMb) || 0,
      }),
    })
      .then(() => {
        notify('resources.podcastChannel.notifications.subscriptionSaved', {
          type: 'info',
        })
        refresh()
      })
      .catch(() => notify('ra.page.error', { type: 'warning' }))
      .finally(() => setSaving(false))
  }

  return (
    <Box mt={2} mb={2} maxWidth={360}>
      <Typography variant="subtitle1" gutterBottom>
        {translate('resources.podcastChannel.subscriptionSettings')}
      </Typography>
      <FormControl fullWidth variant="outlined" margin="dense">
        <InputLabel>
          {translate('resources.podcastChannel.fields.downloadPolicy')}
        </InputLabel>
        <Select
          value={downloadPolicy}
          onChange={(e) => setDownloadPolicy(e.target.value)}
          label={translate('resources.podcastChannel.fields.downloadPolicy')}
        >
          {downloadPolicyChoices.map((id) => (
            <MenuItem key={id} value={id}>
              {translate(`resources.podcastChannel.downloadPolicy.${id}`)}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
      <MuiTextField
        variant="outlined"
        margin="dense"
        fullWidth
        type="number"
        label={translate('resources.podcastChannel.fields.retentionCount')}
        value={retentionCount}
        onChange={(e) => setRetentionCount(e.target.value)}
      />
      <MuiTextField
        variant="outlined"
        margin="dense"
        fullWidth
        type="number"
        label={translate('resources.podcastChannel.fields.retentionDays')}
        value={retentionDays}
        onChange={(e) => setRetentionDays(e.target.value)}
      />
      <MuiTextField
        variant="outlined"
        margin="dense"
        fullWidth
        type="number"
        label={translate('resources.podcastChannel.fields.maxStorageMb')}
        value={maxStorageMb}
        onChange={(e) => setMaxStorageMb(e.target.value)}
      />
      <Box mt={1}>
        <Button variant="outlined" onClick={handleSave} disabled={saving}>
          {translate('ra.action.save')}
        </Button>
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

const ListenedToggle = ({ record }) => {
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const [loading, setLoading] = useState(false)
  if (!record) return null

  const listened = Boolean(record.playCount)
  const stop = (e) => e.stopPropagation()

  const handleClick = (e) => {
    stop(e)
    const toggle = listened
      ? subsonic.markPodcastEpisodeUnlistened
      : subsonic.markPodcastEpisodeListened
    setLoading(true)
    toggle(record.id)
      .then(() => refresh())
      .catch(() => notify('ra.page.error', { type: 'warning' }))
      .finally(() => setLoading(false))
  }

  return (
    <Tooltip
      title={translate(
        listened
          ? 'resources.podcastEpisode.markUnlistened'
          : 'resources.podcastEpisode.markListened',
      )}
    >
      <IconButton
        size="small"
        onClick={handleClick}
        onFocus={stop}
        disabled={loading}
      >
        {listened ? (
          <CheckCircleIcon fontSize="small" color="primary" />
        ) : (
          <CheckCircleOutlineIcon fontSize="small" />
        )}
      </IconButton>
    </Tooltip>
  )
}

const EpisodeActions = ({ record }) => {
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

  // "downloaded" is the CURRENT USER's own "in my list" flag (model.PodcastEpisode.Downloaded) -
  // distinct from downloadStatus, which just tracks whether the shared file exists on disk at all.
  // A file can already exist (downloadStatus=downloaded, fetched for some other subscriber) while
  // this user hasn't personally requested it yet, so they should still see a Download action.
  const isMine = Boolean(record.downloaded)
  const isPending =
    record.downloadStatus === 'downloading' ||
    record.downloadStatus === 'queued'

  return (
    <>
      {/* Only downloaded episodes can be added to a playlist - a playlist entry has no way to
          represent "stream this from the source URL". */}
      {isMine && (
        <IconButton size="small" onClick={handleAddToPlaylist} onFocus={stop}>
          <PlaylistAddIcon fontSize="small" />
        </IconButton>
      )}
      {isMine || isPending ? (
        <IconButton size="small" onClick={handleDelete} onFocus={stop}>
          <DeleteIcon fontSize="small" />
        </IconButton>
      ) : (
        <IconButton size="small" onClick={handleDownload} onFocus={stop}>
          <DownloadIcon fontSize="small" />
        </IconButton>
      )}
    </>
  )
}

const EpisodesSection = ({ channel }) => {
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
          source="playCount"
          label={translate('resources.podcastEpisode.listened')}
          render={(record) => <ListenedToggle record={record} />}
        />
        <FunctionField
          source="id"
          label=""
          sortable={false}
          render={(record) => <EpisodeActions record={record} />}
        />
      </Datagrid>
    </>
  )
}

const PodcastChannelShowLayout = (props) => {
  const { record, loading } = useShowContext(props)
  useResourceRefresh('podcastChannel', 'podcastEpisode')

  if (loading || !record) return null

  return (
    <>
      <RaTitle title={<Title subTitle={record.title} />} />
      <SimpleShowLayout>
        <PodcastChannelHeader />
        <SubscriptionSettings record={record} />
        <ReferenceManyField
          reference="podcastEpisode"
          target="channel_id"
          label=""
          sort={{ field: 'publishDate', order: 'DESC' }}
          perPage={100}
          pagination={<Pagination rowsPerPageOptions={[50, 100, 250]} />}
          fullWidth
        >
          <EpisodesSection channel={record} />
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
