import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useTranslate
} from 'react-admin'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import React from 'react'
import { useDispatch } from 'react-redux'
import { playAlbum } from '../player'

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

  const shuffle = (data) => {
    const ids = Object.keys(data)
    for (let i = ids.length - 1; i > 0; i--) {
      let j = Math.floor(Math.random() * (i + 1))
      ;[ids[i], ids[j]] = [ids[j], ids[i]]
    }
    const shuffled = {}
    ids.forEach((id) => (shuffled[id] = data[id]))
    return shuffled
  }

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <Button
        color={'secondary'}
        onClick={() => {
          dispatch(playAlbum(ids[0], data))
        }}
        label={translate('resources.album.actions.playAll')}
      >
        <PlayArrowIcon />
      </Button>
      <Button
        color={'secondary'}
        onClick={() => {
          const shuffled = shuffle(data)
          const firstId = Object.keys(shuffled)[0]
          dispatch(playAlbum(firstId, shuffled))
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
  onUnselectItems: () => null
}
