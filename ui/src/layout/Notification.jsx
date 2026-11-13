import React from 'react'
import { Notification as RANotification } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'

// RA's primary.light Undo is unreadable on the light snackbar of dark themes
const useStyles = makeStyles(
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
