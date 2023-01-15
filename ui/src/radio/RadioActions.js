import React from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useTranslate,
} from 'react-admin'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import { setTrack } from '../actions'
import { makeStyles } from '@material-ui/core'
import { songFromRadio } from './helper'

const useStyles = makeStyles({
  toolbar: { display: 'flex', justifyContent: 'space-between', width: '100%' },
})

const RadioActions = ({ className, record, ...rest }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const classes = useStyles()

  const newRecord = songFromRadio(record)

  const handlePlay = React.useCallback(() => {
    dispatch(setTrack(newRecord))
  }, [dispatch, newRecord])

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <div className={classes.toolbar}>
        <div>
          <Button
            onClick={handlePlay}
            label={translate('resources.radio.actions.playNow')}
          >
            <PlayArrowIcon />
          </Button>
        </div>
      </div>
    </TopToolbar>
  )
}

RadioActions.propTypes = {
  record: PropTypes.object.isRequired,
}

RadioActions.defaultProps = {
  record: {},
}

export default RadioActions
