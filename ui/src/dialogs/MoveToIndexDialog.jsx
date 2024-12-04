import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { RecordContextProvider, useTranslate } from 'react-admin'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
} from '@material-ui/core'
import { closeMoveToIndexDialog } from '../actions'

/**
 * @component
 * @param {{
 *  title?: string,
 *  onSuccess: (from: number, to: number) => void
 * }}
 */
const MoveToIndexDialog = ({ title, onSuccess }) => {
  const { open, record } = useSelector((state) => state.moveToIndexDialog)
  const dispatch = useDispatch()
  const translate = useTranslate()
  const [to, setTo] = useState(0)

  const handleClose = (e) => {
    dispatch(closeMoveToIndexDialog())
    e.stopPropagation()
  }

  const handleConfirm = (e) => {
    // FIXME: verify: I get why to should be decremented, but why the id? does it start from 0 and displays from 1?
    onSuccess(record.id - 1, parseInt(to) - 1)
    dispatch(closeMoveToIndexDialog())
    e.stopPropagation()
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      aria-labelledby="moveToIndex-dialog-song"
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="moveToIndex-dialog-song">
        {translate(title || 'resources.song.actions.moveToIndex')}
      </DialogTitle>
      <DialogContent>
        {/* TODO: Validate index, min/max and integer test  */}
        <TextField 
            value={to}
            onChange={(e) => setTo(e.target.value)}
        />
        {/* TODO: Preview of songs above/below possible? */}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} color="primary">
          {translate('ra.action.close')}
        </Button>
        <Button onClick={handleConfirm} color="primary">
          {translate('ra.action.confirm')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

MoveToIndexDialog.propTypes = {
  title: PropTypes.string,
  content: PropTypes.object.isRequired,
  onSuccess: PropTypes.func.isRequired
}

export default MoveToIndexDialog
