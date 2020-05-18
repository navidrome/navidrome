import React, { Fragment, useEffect } from 'react'
import { useUnselectAll } from 'react-admin'
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
      <AddToQueueButton {...props} />
      <AddToPlaylistButton {...props} />
    </Fragment>
  )
}
