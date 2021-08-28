/*
 * [Experimental]
 * Custom Hook to handle infinite scrolling data
 * within a list
 */

import { useEffect, useRef } from 'react'
import { useSelector } from 'react-redux'
import { useListContext } from 'ra-core'
import { useDataProvider } from 'react-admin'

export default function useVirtualizedData() {
  const { resource, perPage, currentSort, filterValues } = useListContext()

  const loadedIds = useRef({})

  const { data, ids, total } = useSelector((state) => ({
    ids: state.admin.resources[resource].list.ids,
    data: state.admin.resources[resource].data,
    total: state.admin.resources[resource].list.total,
  }))

  const dataProvider = useDataProvider()

  // ids change only on first mount, or an external trigger
  // like currentSort, filterValue, delete etc
  useEffect(() => {
    loadedIds.current = {}
    for (let i = 0; i < ids.length; i++) {
      loadedIds.current[i] = ids[i]
    }
  }, [ids])

  const handleLoadMore = ({ startIndex, stopIndex }) => {
    // React Admin always fetches the first page for us, so we
    // don't need to fetch it again
    if (startIndex < perPage) return

    const page = Math.floor(startIndex / perPage) + 1

    return dataProvider
      .getList(resource, {
        pagination: { page, perPage },
        sort: currentSort,
        filter: filterValues,
      })
      .then(({ data, total }) => {
        const newStopIndex = Math.min(total - 1, startIndex + perPage - 1)
        for (let i = startIndex; i <= newStopIndex; i++) {
          loadedIds.current[i] = data[i - startIndex].id
        }
      })
  }

  return { data, loadedIds: loadedIds.current, total, handleLoadMore }
}
