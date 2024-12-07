import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { useTranslate } from 'react-admin'
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
 *  onSuccess: (from: number, to: number) => void,
 *  max: number
 * }}
 */
const MoveToIndexDialog = ({ title, onSuccess, max }) => {
  const { open, record } = useSelector((state) => state.moveToIndexDialog)
  const dispatch = useDispatch()
  const translate = useTranslate()
  /**
   * @type {ReturnType<typeof useState<string>>}
   */
  const [to, setTo] = useState("1");
  /**
   * @type {ReturnType<typeof useState<number>>}
   */
  const [validationError, setValidationError] = useState();

  React.useEffect(() => {
    if (!to) {
      setValidationError(translate("ra.validation.required"));
      return;
    }

    const value = parseInt(to);
    if (Number.isNaN(value)) {
      setValidationError(translate("ra.validation.number"));
      return;
    }

    if (value < 1) {
      setValidationError(translate("ra.validation.minValue", { min: 0 }));
      return;
    }

    if (value > max) {
      setValidationError(translate("ra.validation.maxValue", { max: max}));
      return;
    }

    setValidationError(undefined);
  }, [to, max, translate]);


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
        <TextField 
            value={to}
            onChange={(e) => setTo(e.target.value)}
            helperText={validationError ?? `1 - ${max}`}
            error={!!validationError}
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
