import React from 'react'
import {
  Button,
  useDataProvider,
  useTranslate,
  useUnselectAll,
  useNotify,
} from 'react-admin'
import { useDispatch } from 'react-redux'
import { addTrack } from '../audioplayer'
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
        // Add the tracks to the queue in the selection order
        const tracks = response.data.reduce((acc, cur) => {
          acc[cur.id] = cur
          return acc
        }, {})
        selectedIds.forEach((id) => {
          dispatch(addTrack(tracks[id]))
        })
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
