import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useTranslate,
} from 'react-admin'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import React from 'react'
import { useDispatch } from 'react-redux'
import { playAlbum, shuffleAlbum } from '../audioplayer'

export const AlbumActions = ({
  className,
  ids,
  data,
  exporter,
  permanentFilter,
  ...rest
}) => {
  const dispatch = useDispatch()
  const translate = useTranslate()

  // TODO Not sure why data is accumulating tracks from previous plays... Needs investigation. For now, filter out
  // the unwanted tracks
  const filteredData = ids.reduce((acc, id) => {
    acc[id] = data[id]
    return acc
  }, {})

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <Button
        onClick={() => {
          dispatch(playAlbum(ids[0], filteredData))
        }}
        label={translate('resources.album.actions.playAll')}
      >
        <PlayArrowIcon />
      </Button>
      <Button
        onClick={() => {
          dispatch(shuffleAlbum(filteredData))
        }}
        label={translate('resources.album.actions.shuffle')}
      >
        <ShuffleIcon />
      </Button>
    </TopToolbar>
  )
}

AlbumActions.defaultProps = {
  selectedIds: [],
  onUnselectItems: () => null,
}
