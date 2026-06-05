import React from 'react'
import DeleteIcon from '@material-ui/icons/Delete'
import { makeStyles, alpha } from '@material-ui/core/styles'
import clsx from 'clsx'
import {
  useDeleteWithConfirmController,
  Button,
  Confirm,
  useNotify,
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

const DeleteUserButton = (props) => {
  const { resource, record, basePath, className, onClick, ...rest } = props

  const notify = useNotify()
  const redirect = useRedirect()

  const onSuccess = () => {
    notify('resources.user.notifications.deleted')
    redirect('/user')
  }

  const { open, loading, handleDialogOpen, handleDialogClose, handleDelete } =
    useDeleteWithConfirmController({
      resource,
      record,
      basePath,
      onClick,
      onSuccess,
    })

  const classes = useStyles(props)
  return (
    <>
      <Button
        onClick={handleDialogOpen}
        label="ra.action.delete"
        className={clsx('ra-delete-button', classes.deleteButton, className)}
        key="button"
        {...rest}
      >
        <DeleteIcon />
      </Button>
      <Confirm
        isOpen={open}
        loading={loading}
        title="message.delete_user_title"
        content="message.delete_user_content"
        translateOptions={{
          name: record.name,
        }}
        onConfirm={handleDelete}
        onClose={handleDialogClose}
      />
    </>
  )
}

export default DeleteUserButton
