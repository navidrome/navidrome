import React from 'react'
import { useLocation } from 'react-router-dom'
import { useGetOne } from 'react-admin'
import IconButton from '@material-ui/core/IconButton'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import { StarButton } from '../common'

const Placeholder = () => (
  <IconButton>
    <StarBorderIcon disabled={true} />
  </IconButton>
)

const Toolbar = ({ id }) => {
  const location = useLocation()
  const resource = location.pathname.startsWith('/song') ? 'song' : 'albumSong'
  const { data, loading } = useGetOne(resource, id)

  if (loading) {
    return <Placeholder />
  }

  return <StarButton record={data} resource={resource} />
}

const PlayerToolbar = ({ id }) => (
  <>{id ? <Toolbar id={id} /> : <Placeholder />} </>
)

export default PlayerToolbar
