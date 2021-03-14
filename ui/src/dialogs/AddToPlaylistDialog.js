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
  duplicateSongSkip,
  openDuplicateSongWarning,
} from '../actions'
import { SelectPlaylistInput } from './SelectPlaylistInput'
import { httpClient } from '../dataProvider'
import { REST_URL } from '../consts'
import DuplicateSongDialog from './DuplicateSongDialog'

export const AddToPlaylistDialog = () => {
  const {
    open,
    selectedIds,
    onSuccess,
    duplicateSong,
    duplicateIds,
  } = useSelector((state) => state.addToPlaylistDialog)
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const [value, setValue] = useState({})
  const [check, setCheck] = useState(false)
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
        if (tracks) {
          const dupSng = tracks.filter((song) =>
            selectedIds.some((id) => id === song.id)
          )

          if (dupSng.length) {
            const dupIds = dupSng.map((song) => song.id)
            return dispatch(openDuplicateSongWarning(dupIds))
          }
          return setCheck(true)
        }

        setCheck(true)
      })
      .catch((error) => {
        console.error(error)
        notify('ra.page.error', 'warning')
      })
  }

  const handleSubmit = (e) => {
    if (value.id) {
      addToPlaylist(value.id)
    } else {
      createAndAddToPlaylist()
    }
    setCheck(false)
    dispatch(closeAddToPlaylist())
    e.stopPropagation()
  }

  const handleClickClose = (e) => {
    dispatch(closeAddToPlaylist())
    e.stopPropagation()
  }

  const handleChange = (pls) => {
    if (pls.id) {
      checkDuplicateSong(pls.id)
    }
    setValue(pls)
  }

  const handleDuplicateClose = () => {
    dispatch(closeDuplicateSongDialog())
    dispatch(closeAddToPlaylist())
  }
  const handleDuplicateSubmit = () => {
    addToPlaylist(value.id)
    setCheck(false)
    dispatch(closeDuplicateSongDialog())
    dispatch(closeAddToPlaylist())
  }
  const handleSkip = () => {
    const distinctSongs = selectedIds.filter(
      (id) => duplicateIds.indexOf(id) < 0
    )
    dispatch(duplicateSongSkip(distinctSongs))
    setCheck(true)
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
          <Button onClick={handleSubmit} color="primary" disabled={!check}>
            {translate('ra.action.add')}
          </Button>
        </DialogActions>
      </Dialog>
      <DuplicateSongDialog
        open={duplicateSong}
        handleClickClose={handleDuplicateClose}
        handleSubmit={handleDuplicateSubmit}
        handleSkip={handleSkip}
      />
    </>
  )
}
