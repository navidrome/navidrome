import React, { useCallback } from 'react'
import { useLocation } from 'react-router-dom'
import { useGetOne } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import { StarButton, useToggleStar } from '../common'
import { keyMap } from '../hotkeys'

const Placeholder = () => <StarButton disabled={true} resource={'song'} />

const Toolbar = ({ id }) => {
  const location = useLocation()
  const resource = location.pathname.startsWith('/song') ? 'song' : 'albumSong'
  const { data, loading } = useGetOne(resource, id)
  const [toggleStar, toggling] = useToggleStar(resource, data)

  const handlers = {
    TOGGLE_STAR: useCallback(() => toggleStar(), [toggleStar]),
  }

  return (
    <>
      <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
      <StarButton
        record={data}
        resource={resource}
        disabled={loading || toggling}
      />
    </>
  )
}

const PlayerToolbar = ({ id }) => (id ? <Toolbar id={id} /> : <Placeholder />)

export default PlayerToolbar
