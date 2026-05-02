import React from 'react'
import { useState, useCallback } from 'react'
import {
  Edit,
  SimpleForm,
  TextInput,
  useNotify,
  useRefresh,
  useRedirect,
  useMutation,
} from 'react-admin'
import {
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Button,
  TextField,
  CircularProgress,
} from '@material-ui/core'
import httpClient from '../dataProvider/httpClient'

export const SongEditorDialog = ({ songId, onClose }) => {
  const [formData, setFormData] = useState({
    title: '',
    artist: '',
    album: '',
    year: '',
    genre: '',
    trackNumber: '',
  })
  const [loading, setLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [initialLoading, setInitialLoading] = useState(true)

  const notify = useNotify()
  const refresh = useRefresh()

  useMutation(
    {
      type: 'getOne',
      resource: 'song',
      payload: { id: songId },
    },
    {
      onSuccess: (data) => {
        setFormData({
          title: data?.title || '',
          artist: data?.artist || '',
          album: data?.album || '',
          year: data?.year || '',
          genre: data?.genre || '',
          trackNumber: data?.trackNumber || '',
        })
        setInitialLoading(false)
      },
      onError: () => {
        setInitialLoading(false)
        notify('ra.notification.item_not_found', 'warning')
      },
    }
  )

  useCallback((field) => (event) => {
    setFormData((prev) => ({
      ...prev,
      [field]: event.target.value,
    }))
  }, [])

  const handleSave = async () => {
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
      await httpClient(`/api/v1/song/${songId}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      })
      notify('resources.song.notifications.updated', 'info', { smart_count: 1 })
      refresh()
      if (onClose) onClose()
    } catch (error) {
      notify('ra.notification.updated', { type: 'warning' })
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <Dialog open={!!songId} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>Edit Song</DialogTitle>
      <DialogContent>
        {initialLoading ? (
          <CircularProgress />
        ) : (
          <>
            <TextField
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              fullWidth
              variant="outlined"
              label="Title"
              margin="normal"
            />
            <TextField
              value={formData.artist}
              onChange={(e) => setFormData({ ...formData, artist: e.target.value })}
              fullWidth
              variant="outlined"
              label="Artist"
              margin="normal"
            />
            <TextField
              value={formData.album}
              onChange={(e) => setFormData({ ...formData, album: e.target.value })}
              fullWidth
              variant="outlined"
              label="Album"
              margin="normal"
            />
            <TextField
              value={formData.year}
              onChange={(e) => setFormData({ ...formData, year: e.target.value })}
              fullWidth
              variant="outlined"
              label="Year"
              margin="normal"
              type="number"
            />
            <TextField
              value={formData.genre}
              onChange={(e) => setFormData({ ...formData, genre: e.target.value })}
              fullWidth
              variant="outlined"
              label="Genre"
              margin="normal"
            />
            <TextField
              value={formData.trackNumber}
              onChange={(e) => setFormData({ ...formData, trackNumber: e.target.value })}
              fullWidth
              variant="outlined"
              label="Track #"
              margin="normal"
              type="number"
            />
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={isSaving}>
          Cancel
        </Button>
        <Button onClick={handleSave} disabled={isSaving} startIcon={isSaving ? <CircularProgress size={20} /> : null}>
          Save
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default SongEditorDialog