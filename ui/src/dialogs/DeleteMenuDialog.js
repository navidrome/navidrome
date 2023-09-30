import { useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { Button, Dialog, DialogActions, DialogTitle } from '@material-ui/core'
import subsonic from '../subsonic'
import { closeDeleteMenu } from '../actions'

const DeleteMenuDialog = () => {
  const { open, selectedIds } = useSelector((state) => state.deleteMenuDialog)

  const dispatch = useDispatch()
  const translate = useTranslate()

  const handleClose = (e) => {
    dispatch(closeDeleteMenu())
    e.stopPropagation()
  }

  const handleDelete = (e, distinctIds) => {
    const trackIds = Array.isArray(distinctIds) ? distinctIds : selectedIds
    subsonic.remove(trackIds)
    dispatch(closeDeleteMenu())
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
        {translate('message.deleteDialogTitle')}
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
