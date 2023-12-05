import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  TextField,
} from '@material-ui/core'
import { useCallback, useEffect, useState } from 'react'
import { useTranslate } from 'react-admin'

const RadioDialog = ({ open, onClose, setMetadata, title }) => {
  const translate = useTranslate()

  const [fixedTitle, setFixedTitle] = useState(title)
  const [fixedArtist, setFixedArtist] = useState('')

  useEffect(() => {
    setFixedTitle(title)
    setFixedArtist('')
  }, [title])

  const onSave = useCallback(() => {
    if (fixedArtist && fixedTitle) {
      setMetadata({ artist: fixedArtist, title: fixedTitle, fix: true })
    }

    setFixedArtist('')
    onClose()
  }, [fixedArtist, fixedTitle, onClose, setMetadata])

  return (
    <Dialog
      open={open}
      onClose={onClose}
      onBackdropClick={onClose}
      fullWidth
      maxWidth="md"
      aria-labelledby="form-dialog-radio-metadata"
    >
      <DialogTitle id="form-dialog-radio-metadata">
        {translate('resources.radio.message.noArtistNotif')}
      </DialogTitle>
      <DialogContent>
        <DialogContentText></DialogContentText>
        <TextField
          value={fixedTitle}
          fullWidth
          onChange={(evt) => setFixedTitle(evt.target.value)}
          required
          label={translate('resources.radio.fields.title')}
        />
        <TextField
          value={fixedArtist}
          fullWidth
          onChange={(evt) => setFixedArtist(evt.target.value)}
          required
          label={translate('resources.radio.fields.artist')}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} color="primary">
          {translate('ra.action.cancel')}
        </Button>
        <Button
          color="primary"
          data-testid="dialog-radio-save"
          disabled={!fixedArtist || !fixedTitle}
          onClick={onSave}
        >
          {translate('ra.action.save')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default RadioDialog
