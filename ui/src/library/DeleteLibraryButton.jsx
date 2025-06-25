import React from 'react'
import DeleteIcon from '@material-ui/icons/Delete'
import { makeStyles, alpha } from '@material-ui/core/styles'
import clsx from 'clsx'
import {
  useNotify,
  useDeleteWithConfirmController,
  Button,
  Confirm,
  useTranslate,
  useRedirect,
} from 'react-admin'

const useStyles = makeStyles(
  (theme) => ({
    deleteButton: {
      color: theme.palette.error.main,
      '&:hover': {
        backgroundColor: alpha(theme.palette.error.main, 0.12),
        // Reset on mouse devices
        '@media (hover: none)': {
          backgroundColor: 'transparent',
        },
      },
    },
  }),
  { name: 'RaDeleteWithConfirmButton' },
)

const DeleteLibraryButton = ({
  record,
  resource,
  basePath,
  className,
  ...props
}) => {
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

  const classes = useStyles(props)
  return (
    <>
      <Button
        label="ra.action.delete"
        onClick={handleDialogOpen}
        disabled={loading}
        className={clsx('ra-delete-button', classes.deleteButton, className)}
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
