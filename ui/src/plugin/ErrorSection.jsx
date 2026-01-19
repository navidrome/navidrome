import React from 'react'
import { Typography } from '@material-ui/core'
import Alert from '@material-ui/lab/Alert'

export const ErrorSection = ({ error, translate }) => {
  if (!error) return null

  return (
    <Alert severity="error" style={{ marginBottom: 16 }}>
      <Typography variant="subtitle2">
        {translate('resources.plugin.fields.lastError')}
      </Typography>
      <Typography variant="body2">{error}</Typography>
    </Alert>
  )
}
