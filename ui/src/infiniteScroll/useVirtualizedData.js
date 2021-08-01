/*
 * [Experimental]
 * Custom Hook to handle infinite scrolling data
 * within a list
 */

import { useCallback, useEffect, useRef } from 'react'
import { useSelector, useDispatch } from 'react-redux'
import { useListContext, crudGetList } from 'ra-core'

export default function useVirtualizedData() {
  const { resource, perPage, currentSort, filterValues } = useListContext()

  const loadPromiseResolver = useRef(null)
  const loadedIds = useRef({})
  const lastFetchPosition = useRef({
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

  const getList = useCallback(
    (...args) => dispatch(crudGetList(...args)),
    [dispatch]
  )

  useEffect(() => {
    loadedIds.current = {}
    lastFetchPosition.current = { startIndex: 0, stopIndex: perPage }
    getList(resource, { page: 1, perPage }, currentSort, filterValues)
  }, [currentSort, filterValues, getList, resource, perPage])

  useEffect(() => {
    const { startIndex, stopIndex } = lastFetchPosition.current
    for (let i = startIndex; i <= stopIndex; i++) {
      loadedIds.current[i] = ids[i - startIndex]
    }

    lastFetchPosition.current = { startIndex, stopIndex }

    if (loadPromiseResolver.current) {
      loadPromiseResolver.current()
      loadPromiseResolver.current = null
    }
  }, [ids, perPage])

  const handleLoadMore = ({ startIndex, stopIndex }) => {
    const page = Math.floor(startIndex / perPage) + 1
    const newStopIndex = Math.min(total, startIndex + perPage - 1)

    return new Promise((resolve) => {
      loadPromiseResolver.current = resolve
      lastFetchPosition.current = { startIndex, stopIndex: newStopIndex }
      getList(resource, { page: page, perPage }, currentSort, filterValues)
    })
  }

  return { data, loadedIds: loadedIds.current, total, handleLoadMore }
}
