import React, { useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import {
  useCreate,
  useDataProvider,
  useNotify,
  useTranslate,
} from 'react-admin'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from '@material-ui/core'
import {
  closeAddToPlaylist,
  closeDuplicateSongDialog,
  openDuplicateSongWarning,
} from '../actions'
import { SelectPlaylistInput } from './SelectPlaylistInput'
import { httpClient } from '../dataProvider'
import { REST_URL } from '../consts'
import DuplicateSongDialog from './DuplicateSongDialog'

export const AddToPlaylistDialog = () => {
  const { open, selectedIds, onSuccess, duplicateSong } = useSelector(
    (state) => state.addToPlaylistDialog
  )
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const [value, setValue] = useState({})
  const dataProvider = useDataProvider()
  const [createAndAddToPlaylist] = useCreate(
    'playlist',
    { name: value.name },
    {
      onSuccess: ({ data }) => {
        setValue(data)
        addToPlaylist(data.id)
      },
      onFailure: (error) => notify(`Error: ${error.message}`, 'warning'),
    }
  )

  const addToPlaylist = (playlistId) => {
    dataProvider
      .create('playlistTrack', {
        data: { ids: selectedIds },
        filter: { playlist_id: playlistId },
      })
      .then(() => {
        const len = selectedIds.length
        notify('message.songsAddedToPlaylist', 'info', { smart_count: len })
        onSuccess && onSuccess(value, len)
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
  }

  const checkDuplicateSong = (playlistId) => {
    httpClient(`${REST_URL}/playlist/${playlistId}`)
      .then((res) => {
        const { tracks } = JSON.parse(res.body)
        const dupSng = tracks.filter((song) => song.id === selectedIds[0])
        if (dupSng.length) dispatch(openDuplicateSongWarning())
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
  }

  const handleSubmit = (e) => {
    if (value.id) {
      addToPlaylist(value.id)
    } else {
      createAndAddToPlaylist()
    }
    dispatch(closeAddToPlaylist())
    e.stopPropagation()
  }

  const handleClickClose = (e) => {
    dispatch(closeAddToPlaylist())
    e.stopPropagation()
  }

  const handleChange = (pls) => {
     if (pls.id && selectedIds.length === 1) {
       checkDuplicateSong(pls.id)
     }
    setValue(pls)
  }

  const handleDuplicateClose = () => {
    setValue({});
    dispatch(closeDuplicateSongDialog())
  }
  const handleDuplicateSubmit = () => {
    addToPlaylist(value.id)
    dispatch(closeDuplicateSongDialog())
    dispatch(closeAddToPlaylist())
  }

  return (
    <>
      <Dialog
        open={open}
        onClose={handleClickClose}
        onBackdropClick={handleClickClose}
        aria-labelledby="form-dialog-new-playlist"
        fullWidth={true}
        maxWidth={'sm'}
      >
        <DialogTitle id="form-dialog-new-playlist">
          {translate('resources.playlist.actions.selectPlaylist')}
        </DialogTitle>
        <DialogContent>
          <SelectPlaylistInput onChange={handleChange} />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClickClose} color="primary">
            {translate('ra.action.cancel')}
          </Button>
          <Button onClick={handleSubmit} color="primary" disabled={!value.name}>
            {translate('ra.action.add')}
          </Button>
        </DialogActions>
      </Dialog>
      <DuplicateSongDialog
        open={duplicateSong}
        handleClickClose={handleDuplicateClose}
        handleSubmit={handleDuplicateSubmit}
      />
    </>
  )
}
