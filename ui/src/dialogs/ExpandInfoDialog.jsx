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

const ExpandInfoDialog = ({ title, content }) => {
  const { open, record, resource } = useSelector(
    (state) => state.expandInfoDialog,
  )
  const dispatch = useDispatch()
  const translate = useTranslate()
  // A node renders for any record; a map lets one page serve more than one resource.
  const body = React.isValidElement(content) ? content : content?.[resource]

  const handleClose = (e) => {
    dispatch(closeExtendedInfoDialog())
    e.stopPropagation()
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      aria-labelledby="info-dialog-album"
      fullWidth={true}
      maxWidth={'md'}
    >
      <DialogTitle id="info-dialog-album">
        {translate(title || 'resources.song.actions.info')}
      </DialogTitle>
      <DialogContent>
        {record && body && (
          <RecordContextProvider value={record}>{body}</RecordContextProvider>
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
  content: PropTypes.oneOfType([PropTypes.element, PropTypes.object])
    .isRequired,
}

export default ExpandInfoDialog
