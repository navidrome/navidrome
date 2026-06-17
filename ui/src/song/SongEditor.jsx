import React, { useState, useEffect, useCallback } from 'react'
import {
  useGetOne,
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
  TextField,
  CircularProgress,
} from '@material-ui/core'
import httpClient from '../dataProvider/httpClient'

export const SongEditor = ({ songId, song: initialSong, onClose }) => {
  const [song, setSong] = useState(initialSong || null)
  const [formData, setFormData] = useState({
    title: '',
    artist: '',
    album: '',
    year: '',
    genre: '',
    trackNumber: '',
  })
  const [isSaving, setIsSaving] = useState(false)
  const notify = useNotify()
  const translate = useTranslate()
  const refresh = useRefresh()

  const { data: fetchedSong, loading } = useGetOne(
    'song',
    songId,
    { enabled: !!songId && !initialSong }
  )

  useEffect(() => {
    const source = initialSong || fetchedSong
    if (source) {
      setSong(source)
      setFormData({
        title: source.title || '',
        artist: source.artist || '',
        album: source.album || '',
        year: source.year || '',
        genre: source.genre || '',
        trackNumber: source.trackNumber || '',
      })
    }
  }, [initialSong, fetchedSong])

  const handleChange = useCallback((field) => (event) => {
    setFormData((prev) => ({
      ...prev,
      [field]: event.target.value,
    }))
  }, [])

  const handleSave = useCallback(async () => {
    if (!song) return

    setIsSaving(true)
    const payload = {
      title: formData.title,
      artist: formData.artist,
      album: formData.album,
      year: formData.year ? parseInt(formData.year, 10) : null,
      genre: formData.genre,
      trackNumber: formData.trackNumber ? parseInt(formData.trackNumber, 10) : null,
    }

    try {
      await httpClient(`/api/v1/song/${song.id}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      })
      notify('resources.song.notifications.updated', 'info', { smart_count: 1 })
      refresh()
      if (onClose) {
        onClose()
      }
    } catch (error) {
      notify('ra.notification.updated', { type: 'warning' })
    } finally {
      setIsSaving(false)
    }
  }, [song, formData, notify, refresh, onClose])

  const handleClose = useCallback(() => {
    if (!isSaving && onClose) {
      onClose()
    }
  }, [isSaving, onClose])

  const isOpen = !!song

  return (
    <Dialog
      open={isOpen}
      onClose={handleClose}
      aria-labelledby="song-editor-dialog"
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="song-editor-dialog">
        {translate('resources.song.actions.edit', { _: 'Edit Song' })}
      </DialogTitle>
      <DialogContent>
        {loading ? (
          <CircularProgress />
        ) : (
          <>
            <TextField
              value={formData.title}
              onChange={handleChange('title')}
              autoFocus
              fullWidth
              variant={'outlined'}
              label={translate('resources.song.fields.title', { _: 'Title' })}
              disabled={isSaving}
              margin="normal"
            />
            <TextField
              value={formData.artist}
              onChange={handleChange('artist')}
              fullWidth
              variant={'outlined'}
              label={translate('resources.song.fields.artist', { _: 'Artist' })}
              disabled={isSaving}
              margin="normal"
            />
            <TextField
              value={formData.album}
              onChange={handleChange('album')}
              fullWidth
              variant={'outlined'}
              label={translate('resources.song.fields.album', { _: 'Album' })}
              disabled={isSaving}
              margin="normal"
            />
            <TextField
              value={formData.year}
              onChange={handleChange('year')}
              fullWidth
              variant={'outlined'}
              label={translate('resources.song.fields.year', { _: 'Year' })}
              disabled={isSaving}
              margin="normal"
              type="number"
            />
            <TextField
              value={formData.genre}
              onChange={handleChange('genre')}
              fullWidth
              variant={'outlined'}
              label={translate('resources.song.fields.genre', { _: 'Genre' })}
              disabled={isSaving}
              margin="normal"
            />
            <TextField
              value={formData.trackNumber}
              onChange={handleChange('trackNumber')}
              fullWidth
              variant={'outlined'}
              label={translate('resources.song.fields.trackNumber', { _: 'Track #' })}
              disabled={isSaving}
              margin="normal"
              type="number"
            />
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} color="primary" disabled={isSaving}>
          {translate('ra.action.cancel')}
        </Button>
        <Button
          onClick={handleSave}
          color="primary"
          disabled={isSaving}
          startIcon={isSaving ? <CircularProgress size={20} /> : null}
        >
          {translate('ra.action.save')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default SongEditor