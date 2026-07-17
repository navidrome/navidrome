import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useTranslate } from 'react-admin'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  makeStyles,
} from '@material-ui/core'
import { closeTagSongDialog } from '../actions'
import { SelectTagInput } from './SelectTagInput'

const useStyles = makeStyles({
  dialogContent: {
    paddingTop: '0.5em',
    paddingBottom: '0.5em',
  },
})

export const TagSongDialog = () => {
  const classes = useStyles()
  const { open, record } = useSelector((state) => state.tagSongDialog)
  const dispatch = useDispatch()
  const translate = useTranslate()

  const handleClose = (e) => {
    dispatch(closeTagSongDialog())
    e && e.stopPropagation()
  }

  if (!record) {
    return null
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      aria-labelledby="form-dialog-tag-song"
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="form-dialog-tag-song">
        {translate('resources.song.actions.editTags')}
      </DialogTitle>
      <DialogContent className={classes.dialogContent}>
        <SelectTagInput mediaFileId={record.mediaFileId || record.id} />
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} color="primary">
          {translate('ra.action.close')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
