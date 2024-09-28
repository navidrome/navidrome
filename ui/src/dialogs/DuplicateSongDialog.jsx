import React from 'react'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from '@material-ui/core'

import { useTranslate } from 'react-admin'

const DuplicateSongDialog = ({
  open,
  handleClickClose,
  handleSubmit,
  handleSkip,
}) => {
  const translate = useTranslate()

  return (
    <Dialog
      open={open}
      onClose={handleClickClose}
      aria-labelledby="form-dialog-duplicate-song"
    >
      <DialogTitle id="form-dialog-duplicate-song">
        {translate('resources.playlist.message.duplicate_song')}
      </DialogTitle>
      <DialogContent>
        {translate('resources.playlist.message.song_exist')}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClickClose} color="primary">
          {translate('ra.action.cancel')}
        </Button>
        <Button onClick={handleSkip} color="primary">
          {translate('ra.action.skip')}
        </Button>
        <Button onClick={handleSubmit} color="primary">
          {translate('ra.action.add')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default DuplicateSongDialog
