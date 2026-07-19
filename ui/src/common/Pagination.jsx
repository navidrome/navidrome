import React, { useEffect } from 'react'
import {
  Pagination as RAPagination,
  useListPaginationContext,
} from 'react-admin'
import { setStoredPerPage } from './perPageStore'

export const defaultRowsPerPageOptions = [15, 25, 50]

export const Pagination = (props) => {
  const { resource, perPage } = useListPaginationContext()
  useEffect(() => {
    if (resource && perPage) setStoredPerPage(resource, perPage)
  }, [resource, perPage])
  return (
    <RAPagination rowsPerPageOptions={defaultRowsPerPageOptions} {...props} />
  )
}
