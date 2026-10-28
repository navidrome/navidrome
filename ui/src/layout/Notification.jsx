import React from 'react'
import { Notification as RANotification } from 'react-admin'
import { useTheme } from '@mui/material/styles'

// RA's primary.light Undo is unreadable on the light snackbar of dark themes
// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles(
  {
    undo: {
      color: 'inherit',
    },
  },
  { name: 'NDNotification' },
)

const Notification = (props) => {
  const classes = useStyles()
  return (
    <RANotification
      {...props}
      classes={{ undo: classes.undo }}
      anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
    />
  )
}

export default Notification
