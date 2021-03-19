import React from 'react'
import { useSelector } from 'react-redux'
import PropTypes from 'prop-types'
import get from 'lodash.get'
import { IconButton, makeStyles } from '@material-ui/core'
import { useTheme } from '@material-ui/core/styles'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import PauseIcon from '@material-ui/icons/Pause'
import PlayingLight from '../icons/playing-light.gif'
import PlayingDark from '../icons/playing-dark.gif'
import PausedLight from '../icons/paused-light.png'
import PausedDark from '../icons/paused-dark.png'

const useStyles = makeStyles(() => ({
  playBtn: {
    padding: 0,
    marginRight: 'auto',
  },
}))

const Icon = ({ iconClass, paused }) => {
  const theme = useTheme()

  let icon
  if (paused) {
    icon = theme.palette.type === 'light' ? PausedLight : PausedDark
  } else {
    icon = theme.palette.type === 'light' ? PlayingLight : PlayingDark
  }
  return (
    <>
      <img
        src={icon}
        className={iconClass}
        alt={paused ? 'paused' : 'playing'}
      />
    </>
  )
}

Icon.propTypes = {
  iconClass: PropTypes.string,
  paused: PropTypes.bool,
}

const SongPlayIcon = ({
  onClick,
  className,
  record,
  isCurrent,
  iconClass,
  pauseClass,
}) => {
  const classes = useStyles()
  const currentTrack = useSelector((state) => get(state, 'queue.current', {}))
  const currentId = currentTrack.trackId
  const paused = currentTrack.paused
  return (
    <>
      {(isCurrent && !paused) || (currentId === record?.id && !paused) ? (
        <>
          <IconButton
            className={`${classes.playBtn} ${className}`}
            onClick={onClick}
            size="small"
          >
            <PauseIcon />
          </IconButton>
          <Icon iconClass={iconClass} paused={paused} />
        </>
      ) : (
        <>
          <IconButton
            className={`${pauseClass} ${className}`}
            onClick={onClick}
            size="small"
          >
            <PlayArrowIcon />
          </IconButton>
          {isCurrent && <Icon iconClass={iconClass} paused={paused} />}
        </>
      )}
    </>
  )
}

SongPlayIcon.propTypes = {
  record: PropTypes.object,
  onClick: PropTypes.func,
  className: PropTypes.string,
  iconClass: PropTypes.string,
  isCurrent: PropTypes.bool,
  pauseClass: PropTypes.string,
}

SongPlayIcon.defaultProps = {
  record: {},
}

export default SongPlayIcon
