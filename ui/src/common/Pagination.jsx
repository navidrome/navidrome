import React, { useEffect } from 'react'
import {
  Pagination as RAPagination,
  useListPaginationContext,
} from 'react-admin'
import { setStoredPerPage, defaultRowsPerPageOptions } from './perPageStore'

export const Pagination = (props) => {
  const { resource, perPage } = useListPaginationContext()
  useEffect(() => {
    if (resource && perPage) setStoredPerPage(resource, perPage)
  }, [resource, perPage])
  return (
    <RAPagination rowsPerPageOptions={defaultRowsPerPageOptions} {...props} />
  )
}
