import { fade, makeStyles } from '@material-ui/core'
import DeleteIcon from '@material-ui/icons/Delete'
import clsx from 'clsx'
import React from 'react'
import {
  Button,
  Confirm,
  useDeleteWithConfirmController,
  useNotify,
  useRedirect,
} from 'react-admin'

const useStyles = makeStyles(
  (theme) => ({
    deleteButton: {
      color: theme.palette.error.main,
      '&:hover': {
        backgroundColor: fade(theme.palette.error.main, 0.12),
        // Reset on mouse devices
        '@media (hover: none)': {
          backgroundColor: 'transparent',
        },
      },
    },
  }),
  { name: 'RaDeleteWithConfirmButton' }
)

const DeleteRadioButton = (props) => {
  const { resource, record, basePath, className, onClick, ...rest } = props

  const notify = useNotify()
  const redirect = useRedirect()

  const onSuccess = () => {
    notify('resources.radio.notifications.deleted')
    redirect('/radio')
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
        key="button"
        className={clsx('ra-delete-button', classes.deleteButton, className)}
        {...rest}
      >
        <DeleteIcon />
      </Button>
      <Confirm
        isOpen={open}
        loading={loading}
        title="message.delete_radio_title"
        content="message.delete_radio_content"
        translateOptions={{
          name: record.name,
        }}
        onConfirm={handleDelete}
        onClose={handleDialogClose}
      />
    </>
  )
}

export default DeleteRadioButton
