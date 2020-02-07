import React from 'react'
import {
  Button,
  useDataProvider,
  useUnselectAll,
  useTranslate
} from 'react-admin'
import { useDispatch } from 'react-redux'
import { addTrack } from '../player'
import AddToQueueIcon from '@material-ui/icons/AddToQueue'

import Tooltip from '@material-ui/core/Tooltip'

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
    <Button color="secondary" onClick={addToQueue}>
      <Tooltip
        title={translate('resources.song.bulk.addToQueue')}
        placement="right"
      >
        <AddToQueueIcon />
      </Tooltip>
    </Button>
  )
}

export default AddToQueueButton
