import { Button, makeStyles } from '@mui/material'
import PropTypes from 'prop-types'
import React, { useCallback } from 'react'
import { useRecordContext } from 'react-admin'
import { useDispatch } from 'react-redux'
import { setTrack } from '../actions'
import { songFromRadio } from './helper'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'

// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles((theme) => ({
  button: {
    padding: '5px 0px',
    textTransform: 'none',
    marginRight: theme.spacing(1.5),
  },
}))

export const StreamField = (props) => {
  const record = useRecordContext(props)
  const dispatch = useDispatch()
  const classes = useStyles()

  const playTrack = useCallback(
    async (evt) => {
      evt.stopPropagation()
      evt.preventDefault()
      dispatch(setTrack(await songFromRadio(record)))
    },
    [dispatch, record],
  )

  return (
    <Button className={classes.button} onClick={playTrack}>
      <PlayArrowIcon />
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
