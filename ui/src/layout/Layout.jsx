import React, { useCallback } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Layout as RALayout, toggleSidebar } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { HotKeys } from 'react-hotkeys'
import Menu from './Menu'
import AppBar from './AppBar'
import Notification from './Notification'
import useCurrentTheme from '../themes/useCurrentTheme'
import { useSearchRefocus } from '../common'
import {
  LYRICS_SIDEBAR_RESIZING_BODY_CLASS,
  LYRICS_SIDEBAR_TRANSITION_MS,
} from '../audioplayer/lyricsSidebarWidth'

const useStyles = makeStyles((theme) => ({
  root: {
    paddingBottom: (props) => (props.addPadding ? '80px' : 0),
    minWidth: 0,
    transition: `width ${LYRICS_SIDEBAR_TRANSITION_MS}ms cubic-bezier(0.22, 1, 0.36, 1)`,
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
    },
    [`body.${LYRICS_SIDEBAR_RESIZING_BODY_CLASS} &`]: {
      transition: 'none',
    },
    'body.nd-lyrics-sidebar-open &': {
      width: 'calc(100% - var(--nd-lyrics-sidebar-width, 360px))',
    },
  },
}))

const Layout = (props) => {
  const theme = useCurrentTheme()
  const queue = useSelector((state) => state.player?.queue)
  const classes = useStyles({ addPadding: queue.length > 0 })
  const dispatch = useDispatch()
  useSearchRefocus()

  const keyHandlers = {
    TOGGLE_MENU: useCallback(() => dispatch(toggleSidebar()), [dispatch]),
  }

  return (
    <HotKeys handlers={keyHandlers}>
      <RALayout
        {...props}
        className={classes.root}
        menu={Menu}
        appBar={AppBar}
        theme={theme}
        notification={Notification}
      />
    </HotKeys>
  )
}

export default Layout
