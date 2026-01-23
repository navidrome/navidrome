import React, { useState, useCallback } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import {
  useDataProvider,
  useNotify,
  useTranslate,
  useRefresh,
} from 'react-admin'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  CircularProgress,
} from '@material-ui/core'
import { closeSaveQueueDialog } from '../actions'
import { useHistory } from 'react-router-dom'

export const SaveQueueDialog = () => {
  const dispatch = useDispatch()
  const { open } = useSelector((state) => state.saveQueueDialog)
  const queue = useSelector((state) => state.player.queue)
  const [name, setName] = useState('')
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const translate = useTranslate()
  const history = useHistory()
  const [isSaving, setIsSaving] = useState(false)
  const refresh = useRefresh()

  const handleClose = useCallback(
    (e) => {
      setName('')
      dispatch(closeSaveQueueDialog())
      e.stopPropagation()
    },
    [dispatch],
  )

  const handleSave = useCallback(() => {
    setIsSaving(true)
    const ids = queue.map((item) => item.trackId)
    dataProvider
      .create('playlist', { data: { name } })
      .then((res) => {
        const playlistId = res.data.id
        if (ids.length) {
          return dataProvider
            .create('playlistTrack', {
              data: { ids },
              filter: { playlist_id: playlistId },
            })
            .then(() => res)
        }
        return res
      })
      .then((res) => {
        notify('ra.notification.created', {
          type: 'info',
          messageArgs: { smart_count: 1 },
        })
        dispatch(closeSaveQueueDialog())
        refresh()
        history.push(`/playlist/${res.data.id}/show`)
      })
      .catch(() => notify('ra.page.error', { type: 'warning' }))
      .finally(() => setIsSaving(false))
  }, [dataProvider, dispatch, notify, queue, name, history, refresh])

  const handleKeyPress = useCallback(
    (e) => {
      if (e.key === 'Enter' && name.trim() !== '') {
        handleSave()
      }
    },
    [handleSave, name],
  )

  return (
    <Dialog
      open={open}
      onClose={isSaving ? undefined : handleClose}
      aria-labelledby="save-queue-dialog"
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="save-queue-dialog">
        {translate('resources.playlist.actions.saveQueue', { _: 'Save Queue' })}
      </DialogTitle>
      <DialogContent>
        <TextField
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyPress={handleKeyPress}
          autoFocus
          fullWidth
          variant={'outlined'}
          label={translate('resources.playlist.fields.name')}
          disabled={isSaving}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} color="primary" disabled={isSaving}>
          {translate('ra.action.cancel')}
        </Button>
        <Button
          onClick={handleSave}
          color="primary"
          disabled={name.trim() === '' || isSaving}
          data-testid="save-queue-save"
          startIcon={isSaving ? <CircularProgress size={20} /> : null}
        >
          {translate('ra.action.save')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
