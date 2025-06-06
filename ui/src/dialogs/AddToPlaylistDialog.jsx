import React, { useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import {
  useDataProvider,
  useNotify,
  useRefresh,
  useTranslate,
} from 'react-admin'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  makeStyles,
} from '@material-ui/core'
import {
  closeAddToPlaylist,
  closeDuplicateSongDialog,
  openDuplicateSongWarning,
} from '../actions'
import { SelectPlaylistInput } from './SelectPlaylistInput'
import DuplicateSongDialog from './DuplicateSongDialog'
import { httpClient } from '../dataProvider'
import { REST_URL } from '../consts'

const useStyles = makeStyles({
  dialogPaper: {
    height: '26em',
    maxHeight: '26em',
  },
  dialogContent: {
    height: '17.5em',
    overflowY: 'auto',
    paddingTop: '0.5em',
    paddingBottom: '0.5em',
  },
})

export const AddToPlaylistDialog = () => {
  const classes = useStyles()
  const { open, selectedIds, onSuccess, duplicateSong, duplicateIds } =
    useSelector((state) => state.addToPlaylistDialog)
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
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
    if (trackIds.length) {
      dataProvider
        .create('playlistTrack', {
          data: { ids: trackIds },
          filter: { playlist_id: playlistId },
        })
        .then(() => {
          const len = trackIds.length
          notify('message.songsAddedToPlaylist', {
            messageArgs: { smart_count: len },
          })
          onSuccess && onSuccess(value, len)
          refresh()
        })
        .catch(() => {
          notify('ra.page.error', { type: 'warning' })
        })
    } else {
      notify('message.songsAddedToPlaylist', {
        messageArgs: { smart_count: 0 },
      })
    }
  }

  const checkDuplicateSong = (playlistObject) => {
    httpClient(`${REST_URL}/playlist/${playlistObject.id}/tracks`)
      .then((res) => {
        const tracks = res.json
        if (tracks) {
          const dupSng = tracks.filter((song) =>
            selectedIds.some((id) => id === song.mediaFileId),
          )

          if (dupSng.length) {
            const dupIds = dupSng.map((song) => song.mediaFileId)
            dispatch(openDuplicateSongWarning(dupIds))
          }
        }
        setCheck(true)
      })
      .catch(() => {
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
      (id) => duplicateIds.indexOf(id) < 0,
    )
    value.slice(-1).pop().distinctIds = distinctSongs
    dispatch(closeDuplicateSongDialog())
  }

  return (
    <>
      <Dialog
        open={open}
        onClose={handleClickClose}
        aria-labelledby="form-dialog-new-playlist"
        fullWidth={true}
        maxWidth={'sm'}
        classes={{
          paper: classes.dialogPaper,
        }}
      >
        <DialogTitle id="form-dialog-new-playlist">
          {translate('resources.playlist.actions.selectPlaylist')}
        </DialogTitle>
        <DialogContent className={classes.dialogContent}>
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
