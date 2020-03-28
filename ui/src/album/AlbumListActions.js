import React, { cloneElement } from 'react'
import { Button, sanitizeListRestProps, TopToolbar } from 'react-admin'
import { ButtonGroup } from '@material-ui/core'
import ViewHeadlineIcon from '@material-ui/icons/ViewHeadline'
import ViewModuleIcon from '@material-ui/icons/ViewModule'
import { useDispatch, useSelector } from 'react-redux'
import { ALBUM_GRID_MODE, ALBUM_LIST_MODE, selectViewMode } from './albumState'

const AlbumListActions = ({
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
  fullWidth,
  ...rest
}) => {
  const dispatch = useDispatch()
  const albumView = useSelector((state) => state.albumView)

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      {filters &&
        cloneElement(filters, {
          resource,
          showFilter,
          displayedFilters,
          filterValues,
          context: 'button'
        })}
      <ButtonGroup
        variant="text"
        color="primary"
        aria-label="text primary button group"
      >
        <Button
          size="small"
          color={albumView.mode === ALBUM_LIST_MODE ? 'primary' : 'secondary'}
          onClick={() => dispatch(selectViewMode(ALBUM_LIST_MODE))}
        >
          <ViewHeadlineIcon fontSize="inherit" />
        </Button>
        <Button
          size="small"
          color={albumView.mode === ALBUM_GRID_MODE ? 'primary' : 'secondary'}
          onClick={() => dispatch(selectViewMode(ALBUM_GRID_MODE))}
        >
          <ViewModuleIcon fontSize="inherit" />
        </Button>
      </ButtonGroup>
    </TopToolbar>
  )
}

AlbumListActions.defaultProps = {
  selectedIds: [],
  onUnselectItems: () => null
}

export default AlbumListActions
