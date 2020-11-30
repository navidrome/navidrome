import React from 'react'
import { useLocation } from 'react-router-dom'
import { useGetOne } from 'react-admin'
import { StarButton, useToggleStar } from '../common'
import { useHotkeys } from 'react-hotkeys-hook'

const Placeholder = () => <StarButton disabled={true} resource={'song'} />

const Toolbar = ({ id }) => {
  const location = useLocation()
  const resource = location.pathname.startsWith('/song') ? 'song' : 'albumSong'
  const { data, loading } = useGetOne(resource, id)
  const [toggleStar, toggling] = useToggleStar(resource, data)

  useHotkeys(
    's',
    () => {
      toggleStar()
    },
    {},
    [toggleStar]
  )

  return (
    <StarButton
      record={data}
      resource={resource}
      disabled={loading || toggling}
    />
  )
}

const PlayerToolbar = ({ id }) => (id ? <Toolbar id={id} /> : <Placeholder />)

export default PlayerToolbar
