import { makeStyles } from '@material-ui/core/styles'
import React from 'react'
import { useSelector } from 'react-redux'
import get from 'lodash.get'
import playing from '../icons/playing.gif'
import { FunctionField } from 'react-admin'
import PropTypes from 'prop-types'

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
        <img src={playing} className={classes.playingIcon} alt="playing" />
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
