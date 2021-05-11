import React, { useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useDataProvider, useNotify, useTranslate } from 'react-admin'
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
import DuplicateSongDialog from './DuplicateSongDialog'

export const AddToPlaylistDialog = () => {
  const { open, selectedIds, onSuccess, duplicateSong, duplicateIds } =
    useSelector((state) => state.addToPlaylistDialog)
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const [value, setValue] = useState({})
  const [check, setCheck] = useState(false)
  const dataProvider = useDataProvider()
  const createAndAddToPlaylist = (playlistObject) => {
    dataProvider
      .create('playlist', {
        data: { name: playlistObject.name },
      })
      .then((res) => {
        addToPlaylist(res.data.id)
      })
      .catch((error) => notify(`Error: ${error.message}`, 'warning'))
  }

  const addToPlaylist = (playlistId, distinctIds) => {
    const trackIds = Array.isArray(distinctIds) ? distinctIds : selectedIds
    dataProvider
      .create('playlistTrack', {
        data: { ids: trackIds },
        filter: { playlist_id: playlistId },
      })
      .then(() => {
        const len = trackIds.length
        notify('message.songsAddedToPlaylist', 'info', { smart_count: len })
        onSuccess && onSuccess(value, len)
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
  }

  const checkDuplicateSong = (playlistObject) => {
    dataProvider
      .getOne('playlist', { id: playlistObject.id })
      .then((res) => {
        const tracks = res.data.tracks
        if (tracks) {
          const dupSng = tracks.filter((song) =>
            selectedIds.some((id) => id === song.id)
          )

          if (dupSng.length) {
            const dupIds = dupSng.map((song) => song.id)
            dispatch(openDuplicateSongWarning(dupIds))
          }
        }
        setCheck(true)
      })
      .catch((error) => {
        console.error(error)
        notify('ra.page.error', 'warning')
      })
  }

  const handleSubmit = (e) => {
    value.forEach((playlistObject) => {
      if (playlistObject.id) {
        addToPlaylist(playlistObject.id, playlistObject.distinctIds)
      } else {
        createAndAddToPlaylist(playlistObject)
      }
    })
    setCheck(false)
    setValue({})
    dispatch(closeAddToPlaylist())
    e.stopPropagation()
  }

  const handleClickClose = (e) => {
    setCheck(false)
    setValue({})
    dispatch(closeAddToPlaylist())
    e.stopPropagation()
  }

  const handleChange = (pls) => {
    if (!value.length || pls.length > value.length) {
      let newlyAdded = pls.slice(-1).pop()
      if (newlyAdded.id) {
        setCheck(false)
        checkDuplicateSong(newlyAdded)
      } else setCheck(true)
    } else if (pls.length === 0) setCheck(false)
    setValue(pls)
  }

  const handleDuplicateClose = () => {
    dispatch(closeDuplicateSongDialog())
  }
  const handleDuplicateSubmit = () => {
    dispatch(closeDuplicateSongDialog())
  }
  const handleSkip = () => {
    const distinctSongs = selectedIds.filter(
      (id) => duplicateIds.indexOf(id) < 0
    )
    value.slice(-1).pop().distinctIds = distinctSongs
    dispatch(closeDuplicateSongDialog())
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
          <Button
            onClick={handleSubmit}
            color="primary"
            disabled={!check}
            data-testid="playlist-add"
          >
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
