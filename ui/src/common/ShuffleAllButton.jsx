import React from 'react'
import { Button, useDataProvider, useNotify, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import { playTracks } from '../actions'
import PropTypes from 'prop-types'

export const ShuffleAllButton = ({ filters }) => {
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const notify = useNotify()

  const handleOnClick = () => {
    dataProvider
      .getList('song', {
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'random', order: 'ASC' },
        filter: filters,
      })
      .then((res) => {
        const data = {}
        res.data.forEach((song) => {
          data[song.id] = song
        })
        dispatch(playTracks(data))
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
  }

  return (
    <Button
      onClick={handleOnClick}
      label={translate('resources.song.actions.shuffleAll')}
    >
      <ShuffleIcon />
    </Button>
  )
}

ShuffleAllButton.propTypes = {
  filters: PropTypes.object,
}
ShuffleAllButton.defaultProps = {
  filters: {},
}
