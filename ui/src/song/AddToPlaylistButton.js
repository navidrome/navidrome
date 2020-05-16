import React, { useState } from 'react'
import {
  Button,
  useTranslate,
  useUnselectAll,
  useDataProvider,
  useNotify,
} from 'react-admin'
import SelectPlaylistDialog from '../common/SelectPlaylistDialog'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'

const AddToPlaylistButton = ({ resource, selectedIds }) => {
  const [open, setOpen] = useState(false)
  const [selectedValue, setSelectedValue] = useState('')
  const translate = useTranslate()
  const unselectAll = useUnselectAll()
  const notify = useNotify()
  const dataProvider = useDataProvider()

  const handleClickOpen = () => {
    setOpen(true)
  }

  const handleClose = (value) => {
    if (value !== '') {
      dataProvider
        .create('playlistTrack', {
          data: { ids: selectedIds },
          filter: { playlist_id: value },
        })
        .then(() => {
          notify(`Added ${selectedIds.length} songs to playlist`)
        })
        .catch(() => {
          notify('ra.page.error', 'warning')
        })
    }
    setOpen(false)
    setSelectedValue(value)
    unselectAll(resource)
  }

  return (
    <>
      <Button
        color="secondary"
        onClick={handleClickOpen}
        label={translate('resources.song.actions.addToPlaylist')}
      >
        <PlaylistAddIcon />
      </Button>
      <SelectPlaylistDialog
        selectedValue={selectedValue}
        open={open}
        onClose={handleClose}
      />
    </>
  )
}

export default AddToPlaylistButton
