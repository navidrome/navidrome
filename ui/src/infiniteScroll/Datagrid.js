import React, { useEffect, useCallback } from 'react'
import { useSelector, useDispatch } from 'react-redux'
import { useHistory } from 'react-router-dom'
import { crudGetList, useListContext } from 'react-admin'
import { linkToRecord } from 'ra-core'
import VirtualTable from './VirtualTable'
import { useInstance } from './useInstance'
import union from 'lodash.union'
import difference from 'lodash.difference'
function Datagrid(props) {
  const {
    resource,
    basePath,
    setSort,
    perPage,
    currentSort,
    filterValues,
    onToggleItem,
    selectedIds,
    onSelect,
  } = useListContext()

  const { classes, isRowSelectable, rowClick, hasBulkActions } = props

  const [loadedRows, updateLoadedRows] = useInstance({})
  const [lastFetchPosition, updateLastFetchPosition] = useInstance({
    startIndex: 0,
    stopIndex: perPage,
  })

  const { data, ids, total } = useSelector((state) => ({
    ids: state.admin.resources[resource].list.ids,
    data: state.admin.resources[resource].data,
    total: state.admin.resources[resource].list.total,
    loadedOnce: state.admin.resources[resource].list.loadedOnce,
  }))

  const dispatch = useDispatch()

  const history = useHistory()
  const [loadPromiseResolver, updateLoadPromiseResolver] = useInstance(null)

  const getList = (...args) => dispatch(crudGetList(...args))

  useEffect(() => {
    let { startIndex, stopIndex } = lastFetchPosition
    let newLoadedRows = loadedRows

    if (loadPromiseResolver == null) {
      startIndex = 0
      stopIndex = perPage
      newLoadedRows = {}
      // TODO: scrollToPosition(0)
    }
    // console.log('LoadLog', 'Got', startIndex, stopIndex, ids.length)
    for (let i = startIndex; i <= stopIndex; i++) {
      newLoadedRows[i] = data[ids[i - startIndex]]
    }

    updateLoadedRows(newLoadedRows)
    updateLastFetchPosition({ startIndex, stopIndex })

    if (loadPromiseResolver) {
      loadPromiseResolver()
      updateLoadPromiseResolver(null)
    }
  }, [ids])

  const onRowClick = ({ index, rowData: record }) => {
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
      // case 'expand':
      //     handleToggleExpand(event);
      //     return;
      // case 'toggleSelection':
      //     handleToggleSelection(event);
      //     return;
      default:
        if (effect) history.push(effect)
        return
    }
  }
  const [lastSelected, updateLastSelected] = useInstance(null)

  useEffect(() => {
    if (!selectedIds || selectedIds.length === 0) {
      updateLastSelected(null)
    }
  }, [JSON.stringify(selectedIds)])

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
      const lastSelectedIndex = lastSelected
        ? Object.keys(loadedRows).find((i) => loadedRows[i].id === lastSelected)
        : -1
      updateLastSelected(event.target.checked ? id : null)

      if (event.shiftKey && lastSelectedIndex !== -1) {
        const index = ids.indexOf(id)
        const idsBetweenSelections = ids.slice(
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
    [data, ids, isRowSelectable, onSelect, onToggleItem, selectedIds]
  )

  const handleLoadMore = ({ startIndex, stopIndex }) => {
    const page = Math.floor(startIndex / perPage) + 1
    const newStopIndex = Math.min(total, stopIndex + perPage - 1)

    return new Promise((resolve) => {
      updateLoadPromiseResolver(resolve)
      updateLastFetchPosition({ startIndex, stopIndex: newStopIndex })
      getList(resource, { page: page, perPage }, currentSort, filterValues)
    })
  }

  return (
    <VirtualTable
      remoteDataCount={total || 0}
      loadMoreRows={handleLoadMore}
      isRowLoaded={({ index }) => !!loadedRows[index]}
      rowGetter={({ index }) => loadedRows[index] || {}}
      onRowClick={onRowClick}
      classes={classes}
      resource={resource}
      currentSort={currentSort}
      setSort={setSort}
      basePath={basePath}
      onToggleItem={handleToggleItem}
      hasBulkActions={hasBulkActions}
      selectedIds={selectedIds}
    >
      {props.children}
    </VirtualTable>
  )
}

Datagrid.defaultProps = {}

export default Datagrid
