import React, { Fragment, useEffect } from 'react'
import { useUnselectAll } from 'react-admin'
import AddToQueueButton from '../song/AddToQueueButton'

export const AlbumSongBulkActions = (props) => {
  const unselectAll = useUnselectAll()
  useEffect(() => {
    unselectAll('albumSong')
    // eslint-disable-next-line
  }, [])
  return (
    <Fragment>
      <AddToQueueButton {...props} />
    </Fragment>
  )
}
