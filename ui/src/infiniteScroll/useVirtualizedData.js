/*
 * [Experimental]
 * Custom Hook to handle infinite scrolling data
 * within a list
 */

import { useEffect } from 'react'
import { useSelector, useDispatch } from 'react-redux'
import { useInstance } from './useInstance'
import { useListContext, crudGetList } from 'ra-core'

export default function useVirtualizedData() {
  const { resource, perPage, currentSort, filterValues } = useListContext()

  const [loadPromiseResolver, updateLoadPromiseResolver] = useInstance(null)
  const [loadedIds, updateLoadedIds] = useInstance({})

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

  const getList = (...args) => dispatch(crudGetList(...args))

  useEffect(() => {
    let { startIndex, stopIndex } = lastFetchPosition
    let newLoadedIds = loadedIds

    if (loadPromiseResolver == null) {
      startIndex = 0
      stopIndex = perPage
      newLoadedIds = {}
      // TODO: scrollToPosition(0)
    }

    for (let i = startIndex; i <= stopIndex; i++) {
      newLoadedIds[i] = ids[i - startIndex]
    }

    updateLoadedIds(newLoadedIds)
    updateLastFetchPosition({ startIndex, stopIndex })

    if (loadPromiseResolver) {
      loadPromiseResolver()
      updateLoadPromiseResolver(null)
    }
  }, [ids])

  const handleLoadMore = ({ startIndex, stopIndex }) => {
    const page = Math.floor(startIndex / perPage) + 1
    const newStopIndex = Math.min(total, stopIndex + perPage - 1)

    return new Promise((resolve) => {
      updateLoadPromiseResolver(resolve)
      updateLastFetchPosition({ startIndex, stopIndex: newStopIndex })
      getList(resource, { page: page, perPage }, currentSort, filterValues)
    })
  }

  return { data, loadedIds, total, handleLoadMore }
}
