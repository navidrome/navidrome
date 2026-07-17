import React from 'react'
import { Button, useDataProvider, useNotify, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import LabelIcon from '@material-ui/icons/Label'
import PropTypes from 'prop-types'
import { openAddToPlaylist } from '../actions'

export const BindByTagsButton = ({ filters }) => {
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const notify = useNotify()
  const tagName = filters && filters.user_tag

  const handleOnClick = () => {
    dataProvider
      .getList('song', {
        pagination: { page: 1, perPage: -1 },
        sort: { field: 'title', order: 'ASC' },
        filter: { user_tag: tagName, missing: false },
      })
      .then((res) => {
        const ids = res.data.map((song) => song.id)
        if (ids.length === 0) {
          notify('message.songsAddedToPlaylist', {
            messageArgs: { smart_count: 0 },
          })
          return
        }
        dispatch(openAddToPlaylist({ selectedIds: ids }))
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
  }

  return (
    <Button
      onClick={handleOnClick}
      disabled={!tagName}
      label={translate('resources.song.actions.bindByTags')}
    >
      <LabelIcon />
    </Button>
  )
}

BindByTagsButton.propTypes = {
  filters: PropTypes.object,
}
BindByTagsButton.defaultProps = {
  filters: {},
}
