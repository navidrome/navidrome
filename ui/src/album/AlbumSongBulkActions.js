import React, { Fragment, useEffect } from 'react'
import { useUnselectAll } from 'react-admin'
import { playNext, playTracks } from '../audioplayer'
import { RiPlayList2Fill } from 'react-icons/ri'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import { BatchPlayButton } from '../common'
import AddToPlaylistButton from '../song/AddToPlaylistButton'

export const AlbumSongBulkActions = (props) => {
  const unselectAll = useUnselectAll()
  useEffect(() => {
    unselectAll('albumSong')
    // eslint-disable-next-line
  }, [])
  return (
    <Fragment>
      <BatchPlayButton
        {...props}
        action={playTracks}
        label={'resources.song.actions.playNow'}
        icon={<PlayArrowIcon />}
      />
      <BatchPlayButton
        {...props}
        action={playNext}
        label={'resources.song.actions.playNext'}
        icon={<RiPlayList2Fill />}
      />
      <BatchPlayButton {...props} />
      <AddToPlaylistButton {...props} />
    </Fragment>
  )
}
