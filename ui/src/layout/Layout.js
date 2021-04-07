import React, { useCallback } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Layout, toggleSidebar } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { useMediaQuery } from '@material-ui/core'
import { HotKeys } from 'react-hotkeys'
import Menu from './Menu'
import AppBar from './AppBar'
import Notification from './Notification'
import useCurrentTheme from '../themes/useCurrentTheme'

const useStyles = makeStyles({
  root: {
    height: '100vh',
  },
  appFrame: {
    paddingBottom: ({ addPadding }) => (addPadding ? '80px' : 0),
    overflow: 'auto',
  },
  contentWithSidebar: {
    flexGrow: 1,
    height: '100%',
  },
  mainContent: {
    overflow: 'auto',
  },
})

export default (props) => {
  const theme = useCurrentTheme()
  const dispatch = useDispatch()
  const queue = useSelector((state) => state.queue.queue)
  const isDesktop = useMediaQuery('(min-width:600px)')
  const classes = useStyles({ addPadding: queue.length && isDesktop })

  const keyHandlers = {
    TOGGLE_MENU: useCallback(() => dispatch(toggleSidebar()), [dispatch]),
  }

  return (
    <HotKeys handlers={keyHandlers}>
      <Layout
        {...props}
        className={classes.root}
        menu={Menu}
        appBar={AppBar}
        theme={theme}
        notification={Notification}
        classes={{
          appFrame: classes.appFrame,
          contentWithSidebar: classes.contentWithSidebar,
          content: classes.mainContent,
        }}
      />
    </HotKeys>
  )
}
