import React from 'react'
import PropTypes from 'prop-types'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import { IconButton } from '@material-ui/core'
import { useDispatch } from 'react-redux'

const defaultIcon = <PlayArrowIcon fontSize="small" />

const PlayButton = ({ icon = defaultIcon, action, ...rest }) => {
  const dispatch = useDispatch()

  return (
    <IconButton
      onClick={(e) => {
        e.stopPropagation()
        dispatch(action)
      }}
      {...rest}
      size={'small'}
    >
      {icon}
    </IconButton>
  )
}

PlayButton.propTypes = {
  icon: PropTypes.element,
  action: PropTypes.object,
}
export default PlayButton
