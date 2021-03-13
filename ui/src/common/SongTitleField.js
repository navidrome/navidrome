import { makeStyles } from '@material-ui/core/styles'
import React from 'react'
import PropTypes from 'prop-types'
import { FunctionField } from 'react-admin'

const useStyles = makeStyles({
  icon: {
    width: '32px',
    height: '32px',
    verticalAlign: 'text-top',
    marginLeft: '-8px',
    marginTop: '-7px',
    paddingRight: '3px',
  },
  text: {
    verticalAlign: 'text-top',
  },
})

export const SongTitleField = ({ showTrackNumbers, ...props }) => {
  const classes = useStyles()

  const trackName = (r) => {
    const name = r.title
    if (r.trackNumber && showTrackNumbers) {
      return r.trackNumber.toString().padStart(2, '0') + ' ' + name
    }
    return name
  }

  return (
    <>
      <FunctionField
        {...props}
        source="title"
        render={trackName}
        className={classes.text}
      />
    </>
  )
}

SongTitleField.propTypes = {
  record: PropTypes.object,
  showTrackNumbers: PropTypes.bool,
}

SongTitleField.defaultProps = {
  record: {},
  showTrackNumbers: false,
}
