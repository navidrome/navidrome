import React from 'react'
import { useDispatch } from 'react-redux'
import { Button, useTranslate, useUnselectAll } from 'react-admin'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import { openAddToPlaylist } from '../dialogs/dialogState'

const AddToPlaylistButton = ({ resource, selectedIds, onAddToPlaylist }) => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const unselectAll = useUnselectAll()

  const handleClick = () => {
    dispatch(
      openAddToPlaylist({ selectedIds, onSuccess: () => unselectAll(resource) })
    )
  }

  return (
    <Button
      aria-controls="simple-menu"
      aria-haspopup="true"
      onClick={handleClick}
      color="secondary"
      label={translate('resources.song.actions.addToPlaylist')}
    >
      <PlaylistAddIcon />
    </Button>
  )
}

export default AddToPlaylistButton
