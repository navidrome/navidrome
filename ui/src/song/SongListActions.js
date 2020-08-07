import React, { cloneElement } from 'react'
import { useDispatch } from 'react-redux'
import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useDataProvider,
  useTranslate,
  useNotify,
} from 'react-admin'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import { playTracks } from '../audioplayer'

const ShuffleAllButton = () => {
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const notify = useNotify()

  const handleOnClick = () => {
    dataProvider
      .getList('song', {
        pagination: { page: 1, perPage: 200 },
        sort: { field: 'random', order: 'ASC' },
        filter: {},
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

export const SongListActions = ({
  currentSort,
  className,
  resource,
  filters,
  displayedFilters,
  filterValues,
  permanentFilter,
  exporter,
  basePath,
  selectedIds,
  onUnselectItems,
  showFilter,
  maxResults,
  total,
  ids,
  ...rest
}) => {
  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      {filters &&
        cloneElement(filters, {
          resource,
          showFilter,
          displayedFilters,
          filterValues,
          context: 'button',
        })}
      <ShuffleAllButton />
    </TopToolbar>
  )
}

SongListActions.defaultProps = {
  selectedIds: [],
  onUnselectItems: () => null,
}
