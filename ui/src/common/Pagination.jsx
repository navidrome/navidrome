import React, { useEffect } from 'react'
import {
  Pagination as RAPagination,
  useListPaginationContext,
} from 'react-admin'
import { setStoredPerPage, defaultRowsPerPageOptions } from './perPageStore'

export const Pagination = ({
  rowsPerPageOptions = defaultRowsPerPageOptions,
  ...props
}) => {
  const { resource, perPage } = useListPaginationContext()
  // Persist only a real user choice: an actual option in a multi-option
  // selector, never a forced single option or a URL-injected page size.
  const persistable =
    rowsPerPageOptions.length > 1 && rowsPerPageOptions.includes(perPage)
  useEffect(() => {
    if (resource && persistable) setStoredPerPage(resource, perPage)
  }, [resource, perPage, persistable])
  return <RAPagination rowsPerPageOptions={rowsPerPageOptions} {...props} />
}
