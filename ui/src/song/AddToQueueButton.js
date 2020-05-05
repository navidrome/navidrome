import React from 'react'
import {
  Button,
  useDataProvider,
  useTranslate,
  useUnselectAll,
  useNotify,
} from 'react-admin'
import { useDispatch } from 'react-redux'
import { addTracks } from '../audioplayer'
import AddToQueueIcon from '@material-ui/icons/AddToQueue'

const AddToQueueButton = ({ selectedIds }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const unselectAll = useUnselectAll()
  const notify = useNotify()

  const addToQueue = () => {
    dataProvider
      .getMany('song', { ids: selectedIds })
      .then((response) => {
        // Add tracks to a map for easy lookup by ID, needed for the next step
        const tracks = response.data.reduce((acc, cur) => {
          acc[cur.id] = cur
          return acc
        }, {})
        // Add the tracks to the queue in the selection order
        dispatch(addTracks(selectedIds.map((id) => tracks[id])))
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
    unselectAll('song')
  }

  return (
    <Button
      color="secondary"
      onClick={addToQueue}
      label={translate('resources.song.actions.addToQueue')}
    >
      <AddToQueueIcon />
    </Button>
  )
}

export default AddToQueueButton
