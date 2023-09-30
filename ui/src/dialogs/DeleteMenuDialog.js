import { SimpleForm, useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from '@material-ui/core'
import subsonic from '../subsonic'
import { closeDeleteMenu } from '../actions'

const DeleteMenuDialog = () => {
  const { open, record, recordType } = useSelector(
    (state) => state.deleteMenuDialog
  )
  const dispatch = useDispatch()
  const translate = useTranslate()

  const handleClose = (e) => {
    dispatch(closeDeleteMenu())
    e.stopPropagation()
  }

  const handleDelete = (e) => {
    if (record) {
      const id = record.mediaFileId || record.id
      subsonic.delete(id)
      dispatch(closeDeleteMenu())
    }
    e.stopPropagation()
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      aria-labelledby="delete-dialog"
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="delete-dialog">
        {recordType &&
          translate('message.deleteDialogTitle', {
            resource: translate(`resources.${recordType}.name`, {
              smart_count: 1,
            }).toLocaleLowerCase(),
            name: record?.name || record?.title
          })}
      </DialogTitle>
      <DialogActions>
        <Button onClick={handleClose} color="secondary">
          {translate('ra.action.close')}
        </Button>
        <Button onClick={handleDelete} color="primary">
          {translate('ra.action.delete')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default DeleteMenuDialog
