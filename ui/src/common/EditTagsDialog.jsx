import React, { useState } from 'react'
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Grid,
  Typography,
  FormControlLabel,
  Checkbox,
  DialogContentText,
  makeStyles,
} from '@material-ui/core'
import {
  useDataProvider,
  useNotify,
  useRefresh,
  useTranslate,
} from 'react-admin'
import { CoverArtAvatar } from './CoverArtAvatar'
import { ImageUploadOverlay } from './ImageUploadOverlay'
import { httpClient } from '../dataProvider'

const useStyles = makeStyles((theme) => ({
  imageContainer: {
    position: 'relative',
    width: 120,
    height: 120,
    margin: '0 auto 16px auto',
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: theme.shape.borderRadius,
    overflow: 'hidden',
    backgroundColor: theme.palette.background.default,
  },
  avatar: {
    width: '120px !important',
    height: '120px !important',
  },
}))

const EditTagsDialog = ({ record, open, onClose }) => {
  const classes = useStyles()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const refresh = useRefresh()
  const translate = useTranslate()

  const [values, setValues] = useState({
    title: record.title || '',
    artist: record.artist || '',
    album: record.album || '',
    albumArtist: record.albumArtist || '',
    genre: record.genre || '',
    year: record.year ? record.year.toString() : '',
    trackNumber: record.trackNumber ? record.trackNumber.toString() : '',
    disc: record.disc ? record.disc.toString() : '',
    bpm: record.bpm ? record.bpm.toString() : '',
    compilation: record.compilation ? '1' : '0',
    comment: record.comment || '',
  })

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target
    const val = type === 'checkbox' ? (checked ? '1' : '0') : value
    setValues((prev) => ({ ...prev, [name]: val }))
  }

  const handleDeleteArtwork = () => {
    const id = record.mediaFileId || record.id
    httpClient(`/api/song/${id}/artwork`, {
      method: 'POST',
      body: '',
    })
      .then(() => {
        notify('message.coverRemoved')
        setDeleteDialogOpen(false)
        refresh()
      })
      .catch(() => notify('message.coverRemoveError', { type: 'warning' }))
  }

  const handleSubmit = (e) => {
    e.preventDefault()
    const id = record.mediaFileId || record.id
    dataProvider
      .editSongTags(id, values)
      .then(() => {
        return dataProvider.getOne('song', { id: id })
      })
      .then(() => {
        notify('notification.updated', { type: 'info', smart_count: 1 })
        refresh()
        onClose()
      })
      .catch((err) => {
        notify('notification.http_error', { type: 'warning' })
      })
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      onClick={(e) => e.stopPropagation()}
      fullWidth
      maxWidth="sm"
    >
      <DialogTitle>{translate('resources.song.actions.editTags')}</DialogTitle>
      <form onSubmit={handleSubmit}>
        <DialogContent dividers>
          <Grid container spacing={2}>
            <Grid
              item
              xs={12}
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
              }}
            >
              <Typography
                variant="caption"
                color="textSecondary"
                gutterBottom
                style={{ fontWeight: 'bold', textTransform: 'uppercase' }}
              >
                {translate('message.uploadCover')}
              </Typography>
              <div className={classes.imageContainer}>
                <CoverArtAvatar
                  record={record}
                  variant="square"
                  className={classes.avatar}
                />
                <ImageUploadOverlay
                  entityType="song"
                  entityId={record.mediaFileId || record.id}
                  hasUploadedImage={record.hasUploadedImage}
                  onImageChange={refresh}
                />
              </div>
              <Button
                size="small"
                color="secondary"
                onClick={() => setDeleteDialogOpen(true)}
              >
                {translate('message.removeCover')}
              </Button>
            </Grid>
            <Grid item xs={12}>
              <TextField
                name="title"
                value={values.title}
                onChange={handleChange}
                label={translate('resources.song.fields.title')}
                fullWidth
                variant="outlined"
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                name="artist"
                value={values.artist}
                onChange={handleChange}
                label={translate('resources.song.fields.artist')}
                fullWidth
                variant="outlined"
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                name="album"
                value={values.album}
                onChange={handleChange}
                label={translate('resources.song.fields.album')}
                fullWidth
                variant="outlined"
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                name="albumArtist"
                value={values.albumArtist}
                onChange={handleChange}
                label={translate('resources.song.fields.albumArtist')}
                fullWidth
                variant="outlined"
              />
            </Grid>
            <Grid item xs={6}>
              <TextField
                name="genre"
                value={values.genre}
                onChange={handleChange}
                label={translate('resources.song.fields.genre')}
                fullWidth
                variant="outlined"
              />
            </Grid>
            <Grid item xs={6}>
              <TextField
                name="year"
                value={values.year}
                onChange={handleChange}
                label={translate('resources.song.fields.year')}
                fullWidth
                variant="outlined"
              />
            </Grid>
            <Grid item xs={4}>
              <TextField
                name="trackNumber"
                value={values.trackNumber}
                onChange={handleChange}
                label={translate('resources.song.fields.trackNumber')}
                fullWidth
                variant="outlined"
              />
            </Grid>
            <Grid item xs={4}>
              <TextField
                name="disc"
                value={values.disc}
                onChange={handleChange}
                label={translate('resources.song.fields.disc', {
                  discNumber: '#',
                })}
                fullWidth
                variant="outlined"
              />
            </Grid>
            <Grid item xs={4}>
              <TextField
                name="bpm"
                value={values.bpm}
                onChange={handleChange}
                label={translate('resources.song.fields.bpm')}
                fullWidth
                variant="outlined"
              />
            </Grid>
            <Grid item xs={12}>
              <FormControlLabel
                control={
                  <Checkbox
                    name="compilation"
                    checked={values.compilation === '1'}
                    onChange={handleChange}
                    color="primary"
                  />
                }
                label={translate('resources.song.fields.compilation')}
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                name="comment"
                value={values.comment}
                onChange={handleChange}
                label={translate('resources.song.fields.comment')}
                fullWidth
                multiline
                rows={2}
                variant="outlined"
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>{translate('ra.action.cancel')}</Button>
          <Button type="submit" variant="contained" color="primary">
            {translate('ra.action.save')}
          </Button>
        </DialogActions>
      </form>

      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        aria-labelledby="delete-artwork-dialog-title"
      >
        <DialogTitle id="delete-artwork-dialog-title">
          {translate('message.removeCover')}
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {translate('ra.message.are_you_sure')}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>
            {translate('ra.action.cancel')}
          </Button>
          <Button onClick={handleDeleteArtwork} color="secondary" autoFocus>
            {translate('ra.action.confirm')}
          </Button>
        </DialogActions>
      </Dialog>
    </Dialog>
  )
}

export default EditTagsDialog
