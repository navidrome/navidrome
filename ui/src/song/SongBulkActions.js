import React, { Fragment } from 'react'
import AddToQueueButton from './AddToQueueButton'
import AddToPlaylistButton from './AddToPlaylistButton'
import { RiPlayList2Fill } from 'react-icons/ri'
import { playNext } from '../audioplayer'

export const SongBulkActions = (props) => {
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
