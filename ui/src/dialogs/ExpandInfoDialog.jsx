import React from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { RecordContextProvider, useTranslate } from 'react-admin'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from '@material-ui/core'
import { closeExtendedInfoDialog } from '../actions'

const ExpandInfoDialog = ({ title, content, resource }) => {
  const {
    open,
    record,
    resource: openFor,
  } = useSelector((state) => state.expandInfoDialog)
  const dispatch = useDispatch()
  const translate = useTranslate()
  // One page may mount several of these; each claims the resource it was given.
  const mine = !resource || resource === openFor

  const handleClose = (e) => {
    dispatch(closeExtendedInfoDialog())
    e.stopPropagation()
  }

  return (
    <Dialog
      open={open && mine}
      onClose={handleClose}
      aria-labelledby="info-dialog-album"
      fullWidth={true}
      maxWidth={'md'}
    >
      <DialogTitle id="info-dialog-album">
        {translate(title || 'resources.song.actions.info')}
      </DialogTitle>
      <DialogContent>
        {record && mine && (
          <RecordContextProvider value={record}>
            {content}
          </RecordContextProvider>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} color="primary">
          {translate('ra.action.close')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

ExpandInfoDialog.propTypes = {
  title: PropTypes.string,
  content: PropTypes.element.isRequired,
  resource: PropTypes.string,
}

export default ExpandInfoDialog
