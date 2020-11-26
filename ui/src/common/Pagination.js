import React from 'react'
import { Pagination as RAPagination } from 'react-admin'

export const LIST_PER_PAGE_OPTIONS_DEFAULT = [15, 25, 50]

export const Pagination = (props) => (
  <RAPagination rowsPerPageOptions={LIST_PER_PAGE_OPTIONS_DEFAULT} {...props} />
)
