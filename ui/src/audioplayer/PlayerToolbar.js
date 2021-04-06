import React, { useCallback } from 'react'
import { useLocation } from 'react-router-dom'
import { useGetOne } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import { LoveButton, useToggleLove, ToggleButton } from '../common'
import { useSelector } from 'react-redux'
import { keyMap } from '../hotkeys'
import config from '../config'

const Placeholder = ({ enableVisualization }) => {
  return (
    <>
      {config.enableFavourites && (
        <LoveButton disabled={true} resource={'song'} />
      )}
      {enableVisualization && <ToggleButton disabled={true} />}
    </>
  )
}

const Toolbar = ({ id, enableVisualization }) => {
  const location = useLocation()
  const resource = location.pathname.startsWith('/song') ? 'song' : 'albumSong'
  const { data, loading } = useGetOne(resource, id)
  const [toggleLove, toggling] = useToggleLove(resource, data)

  const handlers = {
    TOGGLE_LOVE: useCallback(() => toggleLove(), [toggleLove]),
  }

  return (
    <>
      <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
      {config.enableFavourites && (
        <LoveButton
          record={data}
          resource={resource}
          disabled={loading || toggling}
        />
      )}
      {enableVisualization && <ToggleButton />}
    </>
  )
}

const PlayerToolbar = ({ id }) => {
  const enableVisualization = useSelector(
    (state) => state.settings.visualization
  )

  return id ? (
    <Toolbar id={id} enableVisualization={enableVisualization} />
  ) : (
    <Placeholder enableVisualization={enableVisualization} />
  )
}

export default PlayerToolbar
