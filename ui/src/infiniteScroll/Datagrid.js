import React, { useEffect, useState } from 'react'
import { connect } from 'react-redux'
import { useHistory } from 'react-router-dom'
import { crudGetList } from 'react-admin'
import { linkToRecord } from 'ra-core'
import VirtualTable from './VirtualTable'
function Datagrid(props) {
  const {
    resource,
    data,
    ids,
    basePath,
    currentSort,
    filterValues,
    crudGetList,
    perPage,
  } = props

  const history = useHistory()
  const [loadPromiseResolver, setLoadPromiseResolver] = useState(null)

  useEffect(() => {
    if (loadPromiseResolver != null) {
      loadPromiseResolver()
      setLoadPromiseResolver(null)
    }
  }, [props.loadedOnce, loadPromiseResolver])

  const updateData = () =>
    crudGetList(
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

  const handleLoadMore = () =>
    new Promise((resolve) => {
      setLoadPromiseResolver(resolve)
      updateData()
    })

  return (
    <VirtualTable
      dataSize={ids.length}
      remoteDataCount={props.total || 0}
      loadMoreRows={handleLoadMore}
      isRowLoaded={({ index }) => ids.length > index}
      rowGetter={({ index }) => data[ids[index]]}
      onRowClick={onRowClick}
      {...props}
    >
      {props.children}
    </VirtualTable>
  )
}

Datagrid.defaultProps = {
  ids: [],
  data: {},
  crudGetList: () => null,
  perPage: 10,
  sort: { field: 'id', order: 'ASC' },
  filterValues: {},
}

const mapStateToProps = (state, ownProps) => {
  const { resource } = ownProps

  return {
    ids: state.admin.resources[resource].list.ids,
    data: state.admin.resources[resource].data,
    total: state.admin.resources[resource].list.total,
    loadedOnce: state.admin.resources[resource].list.loadedOnce,
  }
}

export default connect(mapStateToProps, { crudGetList })(Datagrid)
