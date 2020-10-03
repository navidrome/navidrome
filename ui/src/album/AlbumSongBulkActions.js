import React, { Fragment, useEffect } from 'react'
import { useUnselectAll } from 'react-admin'
import { playNext } from '../audioplayer'
import { RiPlayList2Fill } from 'react-icons/ri'
import AddToQueueButton from '../song/AddToQueueButton'
import AddToPlaylistButton from '../song/AddToPlaylistButton'

export const AlbumSongBulkActions = (props) => {
  const unselectAll = useUnselectAll()
  useEffect(() => {
    unselectAll('albumSong')
    // eslint-disable-next-line
  }, [])
  return (
    <Fragment>
      <AddToQueueButton
        {...props}
        action={playNext}
        label={'resources.song.actions.playNext'}
        icon={<RiPlayList2Fill />}
      />
      <AddToQueueButton {...props} />
      <AddToPlaylistButton {...props} />
    </Fragment>
  )
}
