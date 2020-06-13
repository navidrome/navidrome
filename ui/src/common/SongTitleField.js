import { makeStyles } from '@material-ui/core/styles'
import React from 'react'
import PropTypes from 'prop-types'
import { useSelector } from 'react-redux'
import { FunctionField } from 'react-admin'
import get from 'lodash.get'
import { useTheme } from '@material-ui/core/styles'
import PlayingLight from '../icons/playing-light.gif'
import PlayingDark from '../icons/playing-dark.gif'

const useStyles = makeStyles({
  playingIcon: {
    width: '20px',
    height: '20px',
    verticalAlign: 'text-top',
    marginTop: '-2px',
    paddingRight: '3px',
  },
})

const SongTitleField = ({ showTrackNumbers, ...props }) => {
  const theme = useTheme()
  const classes = useStyles()
  const { record } = props
  const currentTrack = useSelector((state) => get(state, 'queue.current', {}))
  const currentId = currentTrack.trackId
  const paused = currentTrack.paused
  const isCurrent =
    currentId &&
    !paused &&
    (currentId === record.id || currentId === record.mediaFileId)

  const trackName = (r) => {
    const name = r.title
    if (r.trackNumber && showTrackNumbers) {
      return r.trackNumber.toString().padStart(2, '0') + ' ' + name
    }
    return name
  }

  return (
    <>
      {isCurrent && (
        <img
          src={theme.palette.type === 'light' ? PlayingLight : PlayingDark}
          className={classes.playingIcon}
          alt="playing"
        />
      )}
      <FunctionField
        {...props}
        source="title"
        render={trackName}
        sortable={false}
      />
    </>
  )
}

SongTitleField.propTypes = {
  record: PropTypes.object,
  showTrackNumbers: PropTypes.bool,
}

export default SongTitleField
