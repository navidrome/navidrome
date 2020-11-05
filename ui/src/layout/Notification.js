import React from 'react'
import { Notification as RANotification } from 'react-admin'

const Notification = (props) => (
  <RANotification
    {...props}
    anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
  />
)

export default Notification
