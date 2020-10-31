import React, { Fragment, useEffect } from 'react'
import { useUnselectAll } from 'react-admin'
import { addTracks, playNext, playTracks } from '../audioplayer'
import { RiPlayList2Fill, RiPlayListAddFill } from 'react-icons/ri'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import { BatchPlayButton } from './index'
import AddToPlaylistButton from './AddToPlaylistButton'

const SongBulkActions = (props) => {
  const unselectAll = useUnselectAll()
  useEffect(() => {
    unselectAll(props.resource)
  }, [unselectAll, props.resource])
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
      <BatchPlayButton
        {...props}
        action={addTracks}
        label={'resources.song.actions.addToQueue'}
        icon={<RiPlayListAddFill />}
      />
      <AddToPlaylistButton {...props} />
    </Fragment>
  )
}

export default SongBulkActions
