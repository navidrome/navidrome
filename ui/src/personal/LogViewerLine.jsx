import React from 'react'
import { makeStyles } from '@material-ui/core/styles'
import { Chip, Tooltip } from '@material-ui/core'
import clsx from 'clsx'

// Define styles for different log levels
const useStyles = makeStyles((theme) => ({
  line: {
    padding: '2px 8px',
    fontFamily: 'monospace',
    fontSize: '0.85rem',
    whiteSpace: 'nowrap',
    display: 'flex',
    alignItems: 'center',
    '&:hover': {
      backgroundColor: theme.palette.action.hover,
    },
  },
  evenLine: {
    backgroundColor: theme.palette.action.hover,
  },
  time: {
    color: theme.palette.text.secondary,
    marginRight: theme.spacing(1),
    flexShrink: 0,
    fontSize: '0.8rem',
  },
  level: {
    padding: '0px 3px',
    borderRadius: '3px',
    marginRight: theme.spacing(1),
    flexShrink: 0,
    fontWeight: 'bold',
    width: '50px',
    textAlign: 'center',
  },
  trace: {
    backgroundColor: theme.palette.grey[300],
    color: theme.palette.getContrastText(theme.palette.grey[300]),
  },
  debug: {
    backgroundColor: theme.palette.info.light,
    color: theme.palette.getContrastText(theme.palette.info.light),
  },
  info: {
    backgroundColor: theme.palette.primary.main,
    color: theme.palette.getContrastText(theme.palette.primary.main),
  },
  warn: {
    backgroundColor: theme.palette.warning.main,
    color: theme.palette.getContrastText(theme.palette.warning.main),
  },
  error: {
    backgroundColor: theme.palette.error.main,
    color: theme.palette.getContrastText(theme.palette.error.main),
  },
  fatal: {
    backgroundColor: theme.palette.error.dark,
    color: theme.palette.getContrastText(theme.palette.error.dark),
    fontWeight: 'bold',
  },
  message: {
    flexGrow: 1,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    marginRight: theme.spacing(1),
  },
  data: {
    display: 'flex',
    flexWrap: 'nowrap',
    overflowX: 'auto',
    '&::-webkit-scrollbar': {
      display: 'none',
    },
  },
  dataChip: {
    height: 20,
    margin: '0 2px',
    cursor: 'pointer',
    fontSize: '0.7rem',
  },
}))

const formatTime = (time) => {
  const date = new Date(time)
  return date.toLocaleTimeString()
}

// Component for a single log line
const LogViewerLine = React.memo(({ index, style, data }) => {
  const classes = useStyles()
  const { logs, onQuickFilter } = data
  const log = logs[index]

  // Handle click on a data field to set a quick filter
  const handleChipClick = (key, value) => {
    onQuickFilter(`${key}:${value}`)
  }

  return (
    <div
      className={clsx(classes.line, { [classes.evenLine]: index % 2 === 0 })}
      style={style}
    >
      <div className={classes.time}>{formatTime(log.time)}</div>
      <div className={clsx(classes.level, classes[log.level])}>
        {log.level.toUpperCase()}
      </div>
      <Tooltip title={log.message}>
        <div
          className={classes.message}
          onClick={() => onQuickFilter(log.message)}
        >
          {log.message}
        </div>
      </Tooltip>
      <div className={classes.data}>
        {Object.entries(log.data).map(([key, value], chipIndex) => (
          <Tooltip title={`${key}: ${value}`} key={chipIndex}>
            <Chip
              label={`${key}:${value}`}
              size="small"
              className={classes.dataChip}
              onClick={() => handleChipClick(key, value)}
            />
          </Tooltip>
        ))}
      </div>
    </div>
  )
})

LogViewerLine.displayName = 'LogViewerLine'

export default LogViewerLine
