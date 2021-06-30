import React, { useEffect, useRef, useState } from 'react'
import { useSelector, useDispatch } from 'react-redux'
import { useHistory } from 'react-router-dom'
import { crudGetList, useListContext } from 'react-admin'
import { linkToRecord } from 'ra-core'
import VirtualTable from './VirtualTable'
function Datagrid(props) {
  const { resource, basePath, currentSort, setSort, filterValues, perPage } =
    useListContext()

  const [loadedRows, setLoadedRows] = useState({});
  const [loadedRowsCount, setLoadedRowsCount] = useState(0);
  const [lastFetchPosition, setLastFetchPosition] = useState({ 
    startIndex: 0, 
    stopIndex: perPage
  })

  const { data, ids, total } = useSelector((state) => ({
    ids: state.admin.resources[resource].list.ids,
    data: state.admin.resources[resource].data,
    total: state.admin.resources[resource].list.total,
    loadedOnce: state.admin.resources[resource].list.loadedOnce,
  }))

  const dispatch = useDispatch()

  const history = useHistory()
  const loadPromiseResolver = useRef(null)

  const getList = (...args) => dispatch(crudGetList(...args))

  useEffect(() => {
    console.log('sort,filter')
    if (loadedRowsCount > 0) {
      setLoadedRowsCount(0);
      setLoadedRows({})
      handleLoadMore({ startIndex: 0, stopIndex: perPage })
    }
  }, [currentSort, filterValues]);
  
  useEffect(() => {
    if (loadPromiseResolver.current != null) {
      const { startIndex, stopIndex } = lastFetchPosition;
      console.log('LoadLog', 'Got', startIndex, stopIndex, ids.length)
      for (let i = startIndex;i <= stopIndex;i++) {
        if (!loadedRows[i])
          loadedRows[i] = data[ids[i - startIndex]]
      }
      loadPromiseResolver.current()
      loadPromiseResolver.current = null
      setLoadedRowsCount(loadedRowsCount + ids.length)
    }
  }, [ids])

  const onRowClick = ({ index, rowData: record }) => {
    const { rowClick } = props
    const id = ids[index]
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
    const page = Math.floor(startIndex/perPage) + 1;

    // if (startIndex >= lastFetchPosition.startIndex && stopIndex <= lastFetchPosition.stopIndex)
    //   return null;

    const newStopIndex = Math.min(total, stopIndex + perPage - 1);

    getList(
      resource,
      { page: page, perPage },
      currentSort,
      filterValues
    )

    return new Promise((resolve) => {
      setLastFetchPosition({ startIndex, stopIndex: newStopIndex });
      loadPromiseResolver.current = resolve
    })
  }

  return (
    <VirtualTable
      dataSize={ids.length || 0}
      remoteDataCount={total || 0}
      loadMoreRows={handleLoadMore}
      isRowLoaded={({ index }) => !!loadedRows[index] }
      // rowGetter={({ index }) => ids.length > index ? data[ids[index]] : { } }
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
