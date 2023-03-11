import React, { useCallback } from 'react'
import { useGetOne } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import { LoveButton, useToggleLove } from '../common'
import { keyMap } from '../hotkeys'

const Placeholder = () => <LoveButton disabled={true} resource={'song'} />

const Toolbar = ({ id }) => {
  const { data, loading } = useGetOne('song', id)
  const [toggleLove, toggling] = useToggleLove('song', data)

  const handlers = {
    TOGGLE_LOVE: useCallback(() => toggleLove(), [toggleLove]),
  }

  return (
    <>
      <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
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
