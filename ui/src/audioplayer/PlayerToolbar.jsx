import React, { useCallback, useState } from 'react'
import { useDispatch } from 'react-redux'
import { useGetOne, useNotify } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import IconButton from '@material-ui/core/IconButton'
import { CircularProgress, useMediaQuery } from '@material-ui/core'
import { RiSaveLine } from 'react-icons/ri'
import { IoIosRadio } from 'react-icons/io'
import { LoveButton, useToggleLove } from '../common'
import { openSaveQueueDialog } from '../actions'
import { addSimilarToQueue } from '../common/playbackActions'
import config from '../config'
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
  const notify = useNotify()
  const { data, loading } = useGetOne('song', id, { enabled: !!id && !isRadio })
  const [toggleLove, toggling] = useToggleLove('song', data)
  const isDesktop = useMediaQuery('(min-width:810px)')
  const classes = useStyles()
  const [mixLoading, setMixLoading] = useState(false)

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

  const handleInstantMix = useCallback(
    async (e) => {
      e.stopPropagation()
      setMixLoading(true)
      notify('message.startingInstantMix', { type: 'info' })
      try {
        await addSimilarToQueue(dispatch, notify, id)
      } catch (err) {
        // eslint-disable-next-line no-console
        console.error('Error starting instant mix:', err)
        notify('ra.page.error', { type: 'warning' })
      } finally {
        setMixLoading(false)
      }
    },
    [dispatch, notify, id],
  )

  const buttonClass = isDesktop ? classes.button : classes.mobileButton
  const listItemClass = isDesktop ? classes.toolbar : classes.mobileListItem

  const instantMixButton = config.enableExternalServices && (
    <IconButton
      size={isDesktop ? 'small' : undefined}
      onClick={handleInstantMix}
      disabled={isRadio || !id || loading || mixLoading}
      data-testid="instant-mix-button"
      className={buttonClass}
    >
      {mixLoading ? (
        <CircularProgress size={isDesktop ? 20 : 18} />
      ) : (
        <IoIosRadio className={!isDesktop ? classes.mobileIcon : undefined} />
      )}
    </IconButton>
  )

  const saveQueueButton = (
    <IconButton
      size={isDesktop ? 'small' : undefined}
      onClick={handleSaveQueue}
      disabled={isRadio}
      data-testid="save-queue-button"
      className={buttonClass}
    >
      <RiSaveLine className={!isDesktop ? classes.mobileIcon : undefined} />
    </IconButton>
  )

  const loveButton = (
    <LoveButton
      record={data}
      resource={'song'}
      size={isDesktop ? undefined : 'inherit'}
      disabled={loading || toggling || !id || isRadio}
      className={buttonClass}
    />
  )

  return (
    <>
      <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
      {isDesktop ? (
        <li className={`${listItemClass} item`}>
          {instantMixButton}
          {saveQueueButton}
          {loveButton}
        </li>
      ) : (
        <>
          <li className={`${listItemClass} item`}>{instantMixButton}</li>
          <li className={`${listItemClass} item`}>{saveQueueButton}</li>
          <li className={`${listItemClass} item`}>{loveButton}</li>
        </>
      )}
    </>
  )
}

export default PlayerToolbar
