import React, { Fragment } from 'react'
import AddToQueueButton from './AddToQueueButton'
import AddToPlaylistButton from './AddToPlaylistButton'

export const SongBulkActions = (props) => {
  return (
    <Fragment>
      <AddToQueueButton {...props} />
      <AddToPlaylistButton {...props} />
    </Fragment>
  )
}
