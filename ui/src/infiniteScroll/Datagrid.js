import React, { useEffect, useRef } from 'react'
import { useSelector, useDispatch } from 'react-redux'
import { useHistory } from 'react-router-dom'
import { crudGetList, useListContext } from 'react-admin'
import { linkToRecord } from 'ra-core'
import VirtualTable from './VirtualTable'
function Datagrid(props) {
  const { resource, basePath, currentSort, setSort, filterValues, perPage } =
    useListContext()

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
  console.log('RenderGrid', Math.random())

  useEffect(() => {
    if (loadPromiseResolver.current != null) {
      loadPromiseResolver.current()
      loadPromiseResolver.current = null
    }
  }, [ids])

  const updateData = () =>
    getList(
      resource,
      { page: 1, perPage: ids.length + perPage },
      currentSort,
      filterValues
    )

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

  const handleLoadMore = () => {
    return new Promise((resolve) => {
      loadPromiseResolver.current = resolve
      updateData()
    })
  }

  return (
    <VirtualTable
      dataSize={ids.length || 0}
      remoteDataCount={total || 0}
      loadMoreRows={handleLoadMore}
      isRowLoaded={({ index }) => ids.length > index}
      rowGetter={({ index }) => data[ids[index]]}
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
