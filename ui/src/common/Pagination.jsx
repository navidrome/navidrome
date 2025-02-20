import React from 'react'
import { Pagination as RAPagination } from 'react-admin'
import { defaultPageSizes } from '../utils/pageSizes'

export const Pagination = (props) => (
  <RAPagination rowsPerPageOptions={defaultPageSizes()} {...props} />
)
