import React, { useEffect, useRef, useState } from 'react'
import { Title, useTranslate, useDataProvider } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import {
  Card,
  CardContent,
  Typography,
  TextField,
  Box,
  Button,
  IconButton,
  Tooltip,
} from '@material-ui/core'
import {
  KeyboardArrowUp as KeyboardArrowUpIcon,
  KeyboardArrowDown as KeyboardArrowDownIcon,
} from '@material-ui/icons'
import FollowIcon from '@material-ui/icons/Visibility'
import FollowOffIcon from '@material-ui/icons/VisibilityOff'
import { baseUrl } from '../utils'
import { REST_URL } from '../consts'
import { FixedSizeList as List } from 'react-window'
import LogViewerLine from './LogViewerLine'
import { useMediaQuery } from '@material-ui/core'

// Define styles for the component
const useStyles = makeStyles((theme) => ({
  root: {
    marginTop: '1em',
  },
  content: {
    padding: 0,
  },
  stickyBar: {
    position: 'sticky',
    top: 0,
    zIndex: 1,
    padding: theme.spacing(1),
    backgroundColor: theme.palette.background.paper,
    display: 'flex',
    alignItems: 'center',
    borderBottom: `1px solid ${theme.palette.divider}`,
  },
  filterInput: {
    marginLeft: theme.spacing(1),
    marginRight: theme.spacing(1),
    flexGrow: 1,
  },
  logContainer: {
    height: 'calc(100vh - 180px)',
    marginTop: theme.spacing(1),
  },
}))

// LogViewer component
const LogViewer = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const isSmall = useMediaQuery((theme) => theme.breakpoints.down('sm'))

  // Local state
  const [logs, setLogs] = useState([])
  const [filter, setFilter] = useState('')
  const [follow, setFollow] = useState(true)
  const [eventSource, setEventSource] = useState(null)
  const listRef = useRef(null)

  // Filter logs
  const filteredLogs = filter
    ? logs.filter((log) => {
        const searchText = filter.toLowerCase()
        return (
          log.message.toLowerCase().includes(searchText) ||
          log.level.toLowerCase().includes(searchText) ||
          Object.entries(log.data).some(
            ([key, value]) =>
              key.toLowerCase().includes(searchText) ||
              String(value).toLowerCase().includes(searchText),
          )
        )
      })
    : logs

  useEffect(() => {
    // Scroll to bottom if follow is enabled
    if (follow && listRef.current && filteredLogs.length > 0) {
      listRef.current.scrollToItem(filteredLogs.length - 1)
    }
  }, [filteredLogs, follow])

  // Connect to log stream
  useEffect(() => {
    const connectToLogStream = () => {
      const token = localStorage.getItem('token')
      if (!token) return

      // URL for log stream endpoint
      const url = baseUrl(`${REST_URL}/admin/logs/stream?jwt=${token}`)

      // Create SSE connection
      const es = new EventSource(url)
      setEventSource(es)

      // Handle incoming log events
      es.onmessage = (event) => {
        const logEntry = JSON.parse(event.data)
        setLogs((prevLogs) => {
          // Keep max 1000 logs
          const newLogs = [...prevLogs, logEntry]
          if (newLogs.length > 1000) {
            return newLogs.slice(newLogs.length - 1000)
          }
          return newLogs
        })
      }

      // Handle errors
      es.onerror = (err) => {
        // eslint-disable-next-line no-console
        console.error('Log stream error:', err)
        es.close()
        // Try to reconnect after a delay
        setTimeout(connectToLogStream, 5000)
      }
    }

    connectToLogStream()

    // Clean up on unmount
    return () => {
      if (eventSource) {
        eventSource.close()
      }
    }
  }, [dataProvider, eventSource])

  // Scroll to top
  const handleScrollTop = () => {
    if (listRef.current && filteredLogs.length > 0) {
      listRef.current.scrollToItem(0)
      setFollow(false)
    }
  }

  // Scroll to bottom
  const handleScrollBottom = () => {
    if (listRef.current && filteredLogs.length > 0) {
      listRef.current.scrollToItem(filteredLogs.length - 1)
    }
  }

  // Toggle follow mode
  const handleToggleFollow = () => {
    setFollow(!follow)
    if (!follow && listRef.current && filteredLogs.length > 0) {
      listRef.current.scrollToItem(filteredLogs.length - 1)
    }
  }

  // Quick filter by clicking on a field
  const handleQuickFilter = (text) => {
    setFilter(text)
  }

  return (
    <Card className={classes.root}>
      <Title title={'Navidrome - ' + translate('menu.personal.logs')} />
      <CardContent className={classes.content}>
        <Box className={classes.stickyBar}>
          <Tooltip title={translate('logViewer.follow')}>
            <IconButton
              color={follow ? 'primary' : 'default'}
              onClick={handleToggleFollow}
            >
              {follow ? <FollowIcon /> : <FollowOffIcon />}
            </IconButton>
          </Tooltip>
          <Tooltip title={translate('logViewer.goTop')}>
            <IconButton onClick={handleScrollTop}>
              <KeyboardArrowUpIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title={translate('logViewer.goBottom')}>
            <IconButton onClick={handleScrollBottom}>
              <KeyboardArrowDownIcon />
            </IconButton>
          </Tooltip>
          <TextField
            className={classes.filterInput}
            variant="outlined"
            margin="dense"
            placeholder={translate('logViewer.filter')}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            InputProps={{
              endAdornment: filter && (
                <Button
                  color="primary"
                  size="small"
                  onClick={() => setFilter('')}
                >
                  {translate('ra.action.clear_input')}
                </Button>
              ),
            }}
          />
        </Box>
        <div className={classes.logContainer}>
          {filteredLogs.length === 0 ? (
            <Box p={2}>
              <Typography align="center" color="textSecondary">
                {translate('logViewer.noLogs')}
              </Typography>
            </Box>
          ) : (
            <List
              ref={listRef}
              height={isSmall ? 400 : 600}
              width="100%"
              itemCount={filteredLogs.length}
              itemSize={30}
              itemData={{
                logs: filteredLogs,
                onQuickFilter: handleQuickFilter,
              }}
            >
              {LogViewerLine}
            </List>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

export default LogViewer
