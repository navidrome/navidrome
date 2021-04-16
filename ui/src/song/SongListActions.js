import React, { cloneElement } from 'react'
import { sanitizeListRestProps, TopToolbar } from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { ShuffleAllButton } from '../common'
import ToggleFieldsMenu from '../common/ToggleFieldsMenu'

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
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.up('xs'))
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
      <ShuffleAllButton filters={filterValues} />
      {isXsmall && <ToggleFieldsMenu resource="song" />}
    </TopToolbar>
  )
}

SongListActions.defaultProps = {
  selectedIds: [],
  onUnselectItems: () => null,
}
