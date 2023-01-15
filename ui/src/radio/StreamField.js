import { Button, makeStyles } from '@material-ui/core'
import PropTypes from 'prop-types'
import React, { useCallback } from 'react'
import { useRecordContext } from 'react-admin'
import { useDispatch } from 'react-redux'
import { setTrack } from '../actions'
import { songFromRadio } from './helper'

const useStyles = makeStyles({
  button: {
    padding: '5px 0px',
    textTransform: 'none',
  },
})

export const StreamField = (props) => {
  const record = useRecordContext(props)
  const dispatch = useDispatch()
  const classes = useStyles()

  const playTrack = useCallback(
    (evt) => {
      evt.stopPropagation()
      dispatch(setTrack(songFromRadio(record)))
    },
    [dispatch, record]
  )

  return (
    <Button className={classes.button} onClick={playTrack}>
      {record.streamUrl}
    </Button>
  )
}

StreamField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
}

StreamField.defaultProps = {
  addLabel: true,
}
