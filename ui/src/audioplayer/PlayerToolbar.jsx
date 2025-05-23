import React, { useCallback } from 'react'
import { useDispatch } from 'react-redux'
import { useGetOne } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import IconButton from '@material-ui/core/IconButton'
import { RiSaveLine } from 'react-icons/ri'
import { LoveButton, useToggleLove } from '../common'
import { openSaveQueueDialog } from '../actions'
import { keyMap } from '../hotkeys'

const Placeholder = () => <LoveButton disabled={true} resource={'song'} />

const Toolbar = ({ id }) => {
  const dispatch = useDispatch()
  const { data, loading } = useGetOne('song', id)
  const [toggleLove, toggling] = useToggleLove('song', data)

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

  return (
    <>
      <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
      <IconButton
        size="small"
        onClick={handleSaveQueue}
        data-testid="save-queue-button"
      >
        <RiSaveLine />
      </IconButton>
      <LoveButton
        record={data}
        resource={'song'}
        disabled={loading || toggling}
      />
    </>
  )
}

const PlayerToolbar = ({ id, isRadio }) =>
  id && !isRadio ? <Toolbar id={id} /> : <Placeholder />

export default PlayerToolbar
