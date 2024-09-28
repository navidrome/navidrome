import React, { useCallback } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Layout as RALayout, toggleSidebar } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { HotKeys } from 'react-hotkeys'
import { HTML5Backend } from 'react-dnd-html5-backend'
import { DndProvider } from 'react-dnd'
import Menu from './Menu'
import AppBar from './AppBar'
import Notification from './Notification'
import useCurrentTheme from '../themes/useCurrentTheme'

const useStyles = makeStyles({
  root: { paddingBottom: (props) => (props.addPadding ? '80px' : 0) },
})

const Layout = (props) => {
  const theme = useCurrentTheme()
  const queue = useSelector((state) => state.player?.queue)
  const classes = useStyles({ addPadding: queue.length > 0 })
  const dispatch = useDispatch()

  const keyHandlers = {
    TOGGLE_MENU: useCallback(() => dispatch(toggleSidebar()), [dispatch]),
  }

  return (
    <DndProvider backend={HTML5Backend}>
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
    </DndProvider>
  )
}

export default Layout
