import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import {
  useCreate,
  useDataProvider,
  useTranslate,
  useNotify,
} from 'react-admin'
import {
  Button,
  TextField,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from '@material-ui/core'
import PropTypes from 'prop-types'
import { closeNewPlaylist } from './dialogState'
import {
  addAlbumToPlaylist,
  addTracksToPlaylist,
} from '../common/AddToPlaylistMenu'

const NewPlaylistDialog = ({ onCancel, onSubmit }) => {
  const { open, albumId, selectedIds } = useSelector(
    (state) => state.newPlaylistDialog
  )
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const [value, setValue] = React.useState('')
  const dataProvider = useDataProvider()
  const [create] = useCreate(
    'playlist',
    { name: value },
    {
      onSuccess: ({ data }) => {
        addToPlaylist(data.id)
      },
      onFailure: (error) => notify(`Error: ${error.message}`, 'warning'),
    }
  )

  const addToPlaylist = (playlistId) => {
    const add = albumId
      ? addAlbumToPlaylist(dataProvider, albumId, playlistId)
      : addTracksToPlaylist(dataProvider, selectedIds, playlistId)

    add
      .then((len) => {
        notify('message.songsAddedToPlaylist', 'info', { smart_count: len })
        onSubmit(value)
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
  }

  const handleSubmit = (e) => {
    create()
    dispatch(closeNewPlaylist())
    e.stopPropagation()
  }

  const handleChange = (e) => {
    setValue(e.target.value)
  }

  const handleClickClose = (e) => {
    onCancel(e)
    dispatch(closeNewPlaylist())
    e.stopPropagation()
  }

  return (
    <div>
      <Dialog
        disableBackdropClick
        open={open}
        onClose={handleClickClose}
        aria-labelledby="form-dialog-new-playlist"
      >
        <DialogTitle id="form-dialog-new-playlist">
          {translate('resources.playlist.actions.newPlaylist')}
        </DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            id="name"
            label={translate('resources.playlist.fields.name')}
            type="text"
            fullWidth
            onChange={handleChange}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClickClose} color="primary">
            {translate('ra.action.cancel')}
          </Button>
          <Button onClick={handleSubmit} color="primary">
            {translate('ra.action.create')}
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  )
}

NewPlaylistDialog.propTypes = {
  onCancel: PropTypes.func,
  onSubmit: PropTypes.func,
}

NewPlaylistDialog.defaultProps = {
  onCancel: () => {},
  onSubmit: () => {},
}

export default NewPlaylistDialog
