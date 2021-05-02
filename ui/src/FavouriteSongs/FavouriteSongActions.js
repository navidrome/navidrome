import React, { cloneElement } from 'react'
import { sanitizeListRestProps, TopToolbar } from 'react-admin'
import { ShuffleAllButton } from '../common'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import { Button, useDataProvider, useNotify, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import { playTracks } from '../actions'

const PlayAllSongs = ({ sort }) => {
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const dispatch = useDispatch()
  const handleClick = () => {
    dataProvider
      .getList('song', {
        pagination: { page: 1, perPage: 200 },
        sort,
        filter: { starred: true },
      })
      .then((res) => {
        const data = res.data.reduce(
          (acc, curr) => ({ ...acc, [curr.id]: curr }),
          {}
        )
        dispatch(playTracks(data))
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
  }

  return (
    <Button
      onClick={handleClick}
      label={translate('resources.favouriteSongs.actions.playAll')}
    >
      <PlayArrowIcon />
    </Button>
  )
}

export const FavouriteSongActions = ({
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
      <PlayAllSongs sort={currentSort} />
      <ShuffleAllButton filters={{ starred: true }} />
    </TopToolbar>
  )
}

FavouriteSongActions.defaultProps = {
  selectedIds: [],
  onUnselectItems: () => null,
}
