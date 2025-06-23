import React, { Fragment, useState } from 'react'
import {
  useNotify,
  useRefresh,
  useDelete,
  Button,
  Confirm,
  useTranslate,
  useRedirect,
} from 'react-admin'
import DeleteIcon from '@material-ui/icons/Delete'

const DeleteLibraryButton = ({ record }) => {
  const [open, setOpen] = useState(false)
  const translate = useTranslate()
  const notify = useNotify()
  const redirect = useRedirect()
  const [deleteOne, { loading }] = useDelete()

  const handleClick = () => setOpen(true)
  const handleDialogClose = () => setOpen(false)

  const handleConfirm = async () => {
    try {
      await deleteOne('library', record.id, record)
      notify('resources.library.notifications.deleted', 'info', {
        smart_count: 1,
      })
      redirect('/library')
    } catch (error) {
      notify(
        typeof error === 'string'
          ? error
          : error.message || 'ra.notification.http_error',
        'warning',
      )
    } finally {
      setOpen(false)
    }
  }

  return (
    <Fragment>
      <Button
        label="ra.action.delete"
        onClick={handleClick}
        disabled={loading}
      >
        <DeleteIcon />
      </Button>
      <Confirm
        isOpen={open}
        loading={loading}
        title={translate('resources.library.name', { smart_count: 1 })}
        content={translate('resources.library.messages.deleteConfirm')}
        onConfirm={handleConfirm}
        onClose={handleDialogClose}
      />
    </Fragment>
  )
}

export default DeleteLibraryButton 