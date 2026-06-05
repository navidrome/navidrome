import React, { useState } from 'react'
import DeleteIcon from '@material-ui/icons/Delete'
import { makeStyles, alpha } from '@material-ui/core/styles'
import clsx from 'clsx'
import {
  Button,
  Confirm,
  useNotify,
  useDeleteMany,
  useRefresh,
  useUnselectAll,
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

const DeleteMissingFilesButton = (props) => {
  const { selectedIds, className, deleteAll = false } = props
  const [open, setOpen] = useState(false)
  const unselectAll = useUnselectAll()
  const refresh = useRefresh()
  const notify = useNotify()

  const ids = deleteAll ? [] : selectedIds
  const [deleteMany, { loading }] = useDeleteMany('missing', ids, {
    onSuccess: () => {
      notify('resources.missing.notifications.removed')
      refresh()
      unselectAll('missing')
    },
    onFailure: (error) =>
      notify('Error: missing files not deleted', { type: 'warning' }),
  })
  const handleClick = () => setOpen(true)
  const handleDialogClose = () => setOpen(false)
  const handleConfirm = () => {
    deleteMany()
    setOpen(false)
  }

  const classes = useStyles(props)

  return (
    <>
      <Button
        onClick={handleClick}
        label={
          deleteAll
            ? 'resources.missing.actions.remove_all'
            : 'ra.action.remove'
        }
        key="button"
        className={clsx('ra-delete-button', classes.deleteButton, className)}
      >
        <DeleteIcon />
      </Button>
      <Confirm
        isOpen={open}
        loading={loading}
        title={
          deleteAll
            ? 'message.remove_all_missing_title'
            : 'message.remove_missing_title'
        }
        content={
          deleteAll
            ? 'message.remove_all_missing_content'
            : 'message.remove_missing_content'
        }
        onConfirm={handleConfirm}
        onClose={handleDialogClose}
      />
    </>
  )
}

export default DeleteMissingFilesButton
