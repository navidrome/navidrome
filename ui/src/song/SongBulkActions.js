import React, { Fragment } from 'react'
import AddToQueueButton from './AddToQueueButton'

export const SongBulkActions = (props) => {
  return (
    <Fragment>
      <AddToQueueButton {...props} />
    </Fragment>
  )
}
