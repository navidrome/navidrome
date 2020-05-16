import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useTranslate,
} from 'react-admin'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import React from 'react'
import { useDispatch } from 'react-redux'
import { playTracks } from '../audioplayer'

const PlaylistActions = ({
  className,
  ids,
  data,
  exporter,
  permanentFilter,
  ...rest
}) => {
  const dispatch = useDispatch()
  const translate = useTranslate()

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <Button
        onClick={() => {
          dispatch(playTracks(data, ids))
        }}
        label={translate('resources.album.actions.playAll')}
      >
        <PlayArrowIcon />
      </Button>
    </TopToolbar>
  )
}

PlaylistActions.defaultProps = {
  selectedIds: [],
  onUnselectItems: () => null,
}

export default PlaylistActions
