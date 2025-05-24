import React, { useCallback } from 'react'
import { useDispatch } from 'react-redux'
import { useGetOne } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import IconButton from '@material-ui/core/IconButton'
import { useMediaQuery } from '@material-ui/core'
import { RiSaveLine } from 'react-icons/ri'
import { LoveButton, useToggleLove } from '../common'
import { openSaveQueueDialog } from '../actions'
import { keyMap } from '../hotkeys'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles((theme) => ({
  toolbar: {
    display: 'flex',
    alignItems: 'center',
    flexGrow: 1,
    justifyContent: 'flex-end',
    gap: '0.5rem',
    listStyle: 'none',
    padding: 0,
    margin: 0,
  },
  mobileListItem: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    listStyle: 'none',
    padding: theme.spacing(0.5),
    margin: 0,
    height: 24,
  },
  button: {
    width: '2.5rem',
    height: '2.5rem',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 0,
  },
  mobileButton: {
    width: 24,
    height: 24,
    padding: 0,
    margin: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: '18px',
  },
  mobileIcon: {
    fontSize: '18px',
    display: 'flex',
    alignItems: 'center',
  },
}))

const PlayerToolbar = ({ id, isRadio }) => {
  const dispatch = useDispatch()
  const { data, loading } = useGetOne('song', id)
  const [toggleLove, toggling] = useToggleLove('song', data)
  const isDesktop = useMediaQuery('(min-width:810px)')
  const classes = useStyles()

  const handlers = {
    TOGGLE_LOVE: useCallback(() => toggleLove(), [toggleLove]),
  }

  const handleSaveQueue = useCallback(
    (e) => {
      dispatch(openSaveQueueDialog())
      e.stopPropagation()
    },
    [dispatch],
  )

  if (isDesktop) {
    return (
      <>
        <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
        <li className={`${classes.toolbar} item`}>
          <IconButton
            size="small"
            onClick={handleSaveQueue}
            disabled={isRadio}
            data-testid="save-queue-button"
            className={classes.button}
          >
            <RiSaveLine />
          </IconButton>
          <LoveButton
            record={data}
            resource={'song'}
            disabled={loading || toggling || !id || isRadio}
            className={classes.button}
          />
        </li>
      </>
    )
  }

  // Mobile layout
  return (
    <>
      <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
      <li className={`${classes.mobileListItem} item`}>
        <IconButton
          onClick={handleSaveQueue}
          disabled={isRadio}
          data-testid="save-queue-button"
          className={classes.mobileButton}
        >
          <RiSaveLine className={classes.mobileIcon} />
        </IconButton>
      </li>
      <li className={`${classes.mobileListItem} item`}>
        <LoveButton
          record={data}
          resource={'song'}
          size="inherit"
          disabled={loading || toggling || !id || isRadio}
          className={classes.mobileButton}
        />
      </li>
    </>
  )
}

export default PlayerToolbar
