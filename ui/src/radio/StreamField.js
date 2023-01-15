import { Button, makeStyles } from '@material-ui/core'
import PropTypes from 'prop-types'
import React, { useCallback } from 'react'
import { useRecordContext } from 'react-admin'
import { useDispatch } from 'react-redux'
import { setTrack } from '../actions'
import { songFromRadio } from './helper'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'

const useStyles = makeStyles((theme) => ({
  button: {
    padding: '5px 0px',
    textTransform: 'none',
    marginRight: theme.spacing(1.5),
  },
}))

export const StreamField = ({ showUrl, ...rest }) => {
  const record = useRecordContext(rest)
  const dispatch = useDispatch()
  const classes = useStyles()

  const playTrack = useCallback(
    async (evt) => {
      evt.stopPropagation()
      dispatch(setTrack(await songFromRadio(record)))
    },
    [dispatch, record]
  )

  return (
    <Button className={classes.button} onClick={playTrack}>
      <PlayArrowIcon />
      {showUrl && record.streamUrl}
    </Button>
  )
}

StreamField.propTypes = {
  label: PropTypes.string,
  record: PropTypes.object,
  source: PropTypes.string.isRequired,
  showUrl: PropTypes.bool,
}

StreamField.defaultProps = {
  addLabel: true,
  showUrl: true,
}
