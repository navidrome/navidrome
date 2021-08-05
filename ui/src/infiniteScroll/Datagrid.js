import React, { useEffect, useCallback, useRef } from 'react'
import { useHistory } from 'react-router-dom'
import { useListContext } from 'react-admin'
import { linkToRecord } from 'ra-core'
import VirtualTable from './VirtualTable'
import union from 'lodash.union'
import difference from 'lodash.difference'
import useVirtualizedData from './useVirtualizedData'
function Datagrid(props) {
  const {
    resource,
    basePath,
    setSort,
    currentSort,
    onToggleItem,
    selectedIds,
    onSelect,
  } = useListContext()

  const { data, loadedIds, total, handleLoadMore } = useVirtualizedData()
  const { classes, isRowSelectable, rowClick, hasBulkActions } = props

  const history = useHistory()

  const onRowClick = ({ index, event, rowData: record }) => {
    const id = record.id
    // onToggleItem(id), from List Context can be used to toggle item in the list
    const effect =
      typeof rowClick === 'function'
        ? rowClick(id, basePath || `/${resource}`, record)
        : rowClick
    switch (effect) {
      case 'edit':
        history.push(linkToRecord(basePath || `/${resource}`, id))
        return
      case 'show':
        history.push(linkToRecord(basePath || `/${resource}`, id, 'show'))
        return
      case 'toggleSelection':
        handleToggleItem(id, event)
        return
      default:
        if (effect) history.push(effect)
        return
    }
  }
  const lastSelected = useRef(null)

  useEffect(() => {
    if (!selectedIds || selectedIds.length === 0) {
      lastSelected.current = null
    }
  }, [selectedIds])

  // const handleSelectAll = useCallback(
  //   event => {
  //       if (event.target.checked) {
  //         const all = ids.concat(
  //             selectedIds.filter(id => !ids.includes(id))
  //         );
  //         onSelect(
  //             isRowSelectable
  //                 ? all.filter(id => isRowSelectable(data[id]))
  //                 : all
  //         );
  //       } else {
  //         onSelect([]);
  //       }
  //   },
  //   [data, ids, onSelect, isRowSelectable, selectedIds]
  // );

  const handleToggleItem = useCallback(
    (id, event) => {
      const lastSelectedIndex = lastSelected.current
        ? Object.keys(loadedIds).find(
            (i) => loadedIds[i] === lastSelected.current
          )
        : -1
      lastSelected.current = event.target.checked ? id : null

      if (event.shiftKey && lastSelectedIndex !== -1) {
        const index = Object.values(loadedIds).indexOf(id)
        const idsBetweenSelections = Object.values(loadedIds).slice(
          Math.min(lastSelectedIndex, index),
          Math.max(lastSelectedIndex, index) + 1
        )

        const newSelectedIds = event.target.checked
          ? union(selectedIds, idsBetweenSelections)
          : difference(selectedIds, idsBetweenSelections)

        onSelect(
          isRowSelectable
            ? newSelectedIds.filter((id) => isRowSelectable(data[id]))
            : newSelectedIds
        )
      } else {
        onToggleItem(id)
      }
    },
    [data, isRowSelectable, onSelect, onToggleItem, selectedIds, loadedIds]
  )

  return (
    <VirtualTable
      remoteDataCount={total || 0}
      loadMoreRows={handleLoadMore}
      isRowLoaded={({ index }) => !!loadedIds[index]}
      rowGetter={({ index }) => data[loadedIds[index]] || {}}
      onRowClick={onRowClick}
      classes={classes}
      resource={resource}
      currentSort={currentSort}
      setSort={setSort}
      basePath={basePath}
      onToggleItem={handleToggleItem}
      hasBulkActions={hasBulkActions}
      selectedIds={selectedIds}
      rowHeight={props.rowHeight}
    >
      {props.children}
    </VirtualTable>
  )
}

Datagrid.defaultProps = {
  rowHeight: 55,
}

export default Datagrid
