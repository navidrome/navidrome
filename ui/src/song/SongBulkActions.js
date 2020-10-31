import React, { Fragment } from 'react'
import { BatchPlayButton } from '../common'
import AddToPlaylistButton from './AddToPlaylistButton'
import { RiPlayList2Fill } from 'react-icons/ri'
import { playNext } from '../audioplayer'

export const SongBulkActions = (props) => {
  return (
    <Fragment>
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
