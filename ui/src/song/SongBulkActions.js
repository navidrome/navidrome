import React, { Fragment, useEffect } from 'react'
import { useUnselectAll } from 'react-admin'
import AddToQueueButton from './AddToQueueButton'

export const SongBulkActions = (props) => {
  const unselectAll = useUnselectAll()
  useEffect(() => {
    console.log('UNSELECT!')
    unselectAll('song')
  }, [])
  return (
    <Fragment>
      <AddToQueueButton {...props} />
    </Fragment>
  )
}
