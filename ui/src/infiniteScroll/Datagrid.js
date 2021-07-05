import React, { useEffect } from 'react'
import { useSelector, useDispatch } from 'react-redux'
import { useHistory } from 'react-router-dom'
import { crudGetList, useListContext } from 'react-admin'
import { linkToRecord } from 'ra-core'
import VirtualTable from './VirtualTable'
import { useInstance } from './useInstance'
function Datagrid(props) {
  const { resource, basePath, setSort, perPage, currentSort, filterValues } =
    useListContext()

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
    const { rowClick } = props

    const id = record.id
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
      classes={props.classes}
      resource={resource}
      currentSort={currentSort}
      setSort={setSort}
      basePath={basePath}
    >
      {props.children}
    </VirtualTable>
  )
}

Datagrid.defaultProps = {}

export default Datagrid
