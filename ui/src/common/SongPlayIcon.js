import React from 'react'
import { useSelector } from 'react-redux'
import PropTypes from 'prop-types'
import get from 'lodash.get'
import { IconButton } from '@material-ui/core'
import PlayCircleOutlineIcon from '@material-ui/icons/PlayCircleOutline'
import PauseCircleOutlineIcon from '@material-ui/icons/PauseCircleOutline'

const SongPlayIcon = ({ record, onClick }) => {
  const currentTrack = useSelector((state) => get(state, 'queue.current', {}))
  const currentId = currentTrack.trackId
  const paused = currentTrack.paused
  return (
    <>
      <IconButton onClick={onClick}>
        {currentId === record.id && !paused ? (
          <PauseCircleOutlineIcon />
        ) : (
          <PlayCircleOutlineIcon />
        )}
      </IconButton>
    </>
  )
}

SongPlayIcon.propTypes = {
  record: PropTypes.object,
  onClick: PropTypes.func,
}

SongPlayIcon.defaultProps = {
  record: {},
}

export default SongPlayIcon
