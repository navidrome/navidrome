import React, { useCallback } from 'react'
import { useGetOne } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import { LoveButton, useToggleLove } from '../common'
import { keyMap } from '../hotkeys'
import config from '../config'
import { UpdateQueueButton } from '../common/UpdateQueueButton'

const Placeholder = () => {
  return (
    <>
      {config.enableFavourites && (
        <LoveButton disabled={true} resource={'song'} />
      )}
      <UpdateQueueButton label={'queue'} />
    </>
  )
}

const GetSongId = (data) => {
  let songIDs = []
  for (var i = 0; i < data.length; i++) songIDs.push(data[i].id)
  return songIDs
}

export { GetSongId }
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
      <UpdateQueueButton label="queue" />
    </>
  )
}

const PlayerToolbar = ({ id, isRadio }) =>
  id && !isRadio ? <Toolbar id={id} /> : <Placeholder />

export default PlayerToolbar
