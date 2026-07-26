import React, { useCallback } from 'react'
import {
  Pagination as RAPagination,
  useListPaginationContext,
} from 'react-admin'
import { setStoredPerPage, defaultRowsPerPageOptions } from './perPageStore'

export const Pagination = ({
  rowsPerPageOptions = defaultRowsPerPageOptions,
  ...props
}) => {
  const { resource, setPerPage } = useListPaginationContext()
  // Persist only a selector-driven change: mount, URL params and responsive
  // fallbacks never call setPerPage, so they can't overwrite the preference.
  const handleSetPerPage = useCallback(
    (value) => {
      if (resource) setStoredPerPage(resource, value)
      setPerPage(value)
    },
    [resource, setPerPage],
  )
  return (
    <RAPagination
      rowsPerPageOptions={rowsPerPageOptions}
      {...props}
      setPerPage={handleSetPerPage}
    />
  )
}
