import React from 'react'
import {
  useNotify,
  useDeleteWithConfirmController,
  Button,
  Confirm,
  useTranslate,
  useRedirect,
} from 'react-admin'
import DeleteIcon from '@material-ui/icons/Delete'

const DeleteLibraryButton = ({ record, resource, basePath, ...props }) => {
  const translate = useTranslate()
  const notify = useNotify()
  const redirect = useRedirect()

  const onSuccess = () => {
    notify('resources.library.notifications.deleted', 'info', {
      smart_count: 1,
    })
    redirect('/library')
  }

  const { open, loading, handleDialogOpen, handleDialogClose, handleDelete } =
    useDeleteWithConfirmController({
      resource,
      record,
      basePath,
      onSuccess,
    })

  return (
    <>
      <Button
        label="ra.action.delete"
        onClick={handleDialogOpen}
        disabled={loading}
        {...props}
      >
        <DeleteIcon />
      </Button>
      <Confirm
        isOpen={open}
        loading={loading}
        title={translate('resources.library.name', { smart_count: 1 })}
        content={translate('resources.library.messages.deleteConfirm')}
        onConfirm={handleDelete}
        onClose={handleDialogClose}
      />
    </>
  )
}

export default DeleteLibraryButton 