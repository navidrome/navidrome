import React from 'react'
import { useDispatch } from 'react-redux'
import {
  Button,
  TopToolbar,
  useTranslate,
} from 'react-admin'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import {
  playTracks,
  shuffleTracks,
} from '../actions'

const FolderActions = ({
  ids,
  data,
  ...rest
}) => {
  const dispatch = useDispatch()
  const translate = useTranslate()

  const handlePlay = React.useCallback(() => {
    dispatch(playTracks(data, ids))
  }, [dispatch, data, ids])

  const handleShuffle = React.useCallback(() => {
    dispatch(shuffleTracks(data, ids))
  }, [dispatch, data, ids])

  if (!ids || ids.length === 0) return null

  return (
    <TopToolbar {...rest}>
      <Button
        onClick={handlePlay}
        label={translate('resources.album.actions.playAll')}
      >
        <PlayArrowIcon />
      </Button>
      <Button
        onClick={handleShuffle}
        label={translate('resources.album.actions.shuffle')}
      >
        <ShuffleIcon />
      </Button>
    </TopToolbar>
  )
}

export default FolderActions
