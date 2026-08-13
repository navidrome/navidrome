import React from 'react'
import { useDispatch } from 'react-redux'
import { IconButton, CircularProgress } from '@material-ui/core'
import GetAppIcon from '@material-ui/icons/GetApp'
import DeleteIcon from '@material-ui/icons/Delete'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import subsonic from '../subsonic'
import { setTrack } from '../actions'

const EpisodeActions = ({ episode, onRefresh, channelTitle }) => {
  const dispatch = useDispatch()

  const handleDownload = async () => {
    await subsonic.downloadPodcastEpisode(episode.id)
    onRefresh?.()
  }

  const handleDelete = async (e) => {
    e.stopPropagation()
    await subsonic.deletePodcastEpisode(episode.id)
    onRefresh?.()
  }

  const handlePlay = (e) => {
    e.stopPropagation()
    dispatch(
      setTrack({
        id: episode.streamId,
        title: episode.title,
        album: channelTitle || episode.channelId,
        artist: '',
        duration: episode.duration,
        suffix: episode.suffix,
        isPodcast: true,
        channelId: episode.channelId,
      }),
    )
  }

  if (episode.status === 'downloading') {
    return <CircularProgress size={20} />
  }

  if (episode.status === 'completed') {
    return (
      <>
        <IconButton aria-label="play" size="small" onClick={handlePlay}>
          <PlayArrowIcon fontSize="small" />
        </IconButton>
        <IconButton aria-label="delete" size="small" onClick={handleDelete}>
          <DeleteIcon fontSize="small" />
        </IconButton>
      </>
    )
  }

  if (episode.status === 'new' || episode.status === 'error') {
    return (
      <>
        <IconButton aria-label="download" size="small" onClick={handleDownload}>
          <GetAppIcon fontSize="small" />
        </IconButton>
        <IconButton aria-label="delete" size="small" onClick={handleDelete}>
          <DeleteIcon fontSize="small" />
        </IconButton>
      </>
    )
  }

  return null
}

export default EpisodeActions
