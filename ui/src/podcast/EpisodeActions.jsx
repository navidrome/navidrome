import React from 'react'
import { IconButton, CircularProgress } from '@material-ui/core'
import GetAppIcon from '@material-ui/icons/GetApp'
import DeleteIcon from '@material-ui/icons/Delete'
import subsonic from '../subsonic'

const EpisodeActions = ({ episode, onRefresh }) => {

  const handleDownload = async () => {
    await subsonic.downloadPodcastEpisode(episode.id)
    onRefresh?.()
  }

  const handleDelete = async () => {
    await subsonic.deletePodcastEpisode(episode.id)
    onRefresh?.()
  }

  if (episode.status === 'downloading') {
    return <CircularProgress size={20} />
  }

  if (episode.status === 'completed') {
    return (
      <IconButton aria-label="delete" size="small" onClick={handleDelete}>
        <DeleteIcon fontSize="small" />
      </IconButton>
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
