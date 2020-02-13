import React from 'react'
import {
  Button,
  useDataProvider,
  useTranslate,
  useUnselectAll
} from 'react-admin'
import { useDispatch } from 'react-redux'
import { addTrack } from '../player'
import AddToQueueIcon from '@material-ui/icons/AddToQueue'

const AddToQueueButton = ({ selectedIds }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const unselectAll = useUnselectAll()
  const addToQueue = () => {
    selectedIds.forEach((id) => {
      dataProvider.getOne('song', { id }).then((response) => {
        dispatch(addTrack(response.data))
      })
    })
    unselectAll('song')
  }

  return (
    <Button
      color="secondary"
      onClick={addToQueue}
      label={translate('resources.song.bulk.addToQueue')}
    >
      <AddToQueueIcon />
    </Button>
  )
}

export default AddToQueueButton
