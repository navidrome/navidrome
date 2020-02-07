import React from 'react'
import PropTypes from 'prop-types'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import { IconButton } from '@material-ui/core'
import { useDispatch } from 'react-redux'
import { setTrack } from '../player'

const defaultIcon = <PlayArrowIcon fontSize="small" />

const PlayButton = ({
  record,
  icon = defaultIcon,
  action = setTrack,
  ...rest
}) => {
  const dispatch = useDispatch()

  return (
    <IconButton
      onClick={() => dispatch(action(record))}
      {...rest}
      size={'small'}
    >
      {icon}
    </IconButton>
  )
}

PlayButton.propTypes = {
  record: PropTypes.any,
  icon: PropTypes.element,
  action: PropTypes.func
}
export default PlayButton
