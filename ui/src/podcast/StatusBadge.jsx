import React from 'react'
import { Chip, CircularProgress, Tooltip, Typography, Box, makeStyles } from '@material-ui/core'
import { useTranslate } from 'react-admin'

const useStyles = makeStyles((theme) => ({
  completed: { backgroundColor: theme.palette.success?.main || '#4caf50', color: '#fff' },
  error: { backgroundColor: theme.palette.error.main, color: '#fff', cursor: 'default' },
  new: {},
  skipped: { backgroundColor: theme.palette.warning?.main || '#ff9800', color: '#fff' },
}))

const StatusBadge = ({ status, errorMessage, downloadedBytes, size }) => {
  const translate = useTranslate()
  const classes = useStyles()

  if (!status || status === 'deleted') return null

  const label = translate(`resources.podcast.status.${status}`, { _: status })

  if (status === 'downloading') {
    const pct = size > 0 && downloadedBytes > 0 ? Math.round((downloadedBytes / size) * 100) : null
    return (
      <Box display="flex" alignItems="center" style={{ gap: 6 }}>
        <CircularProgress size={14} />
        {pct !== null && <Typography variant="caption">{`${pct}%`}</Typography>}
      </Box>
    )
  }

  if (status === 'error' && errorMessage) {
    return (
      <Tooltip title={errorMessage}>
        <Chip className={classes.error} label={label} size="small" />
      </Tooltip>
    )
  }

  return (
    <Chip
      className={classes[status] || classes.new}
      label={label}
      size="small"
    />
  )
}

export default StatusBadge
