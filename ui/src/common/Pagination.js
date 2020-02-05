import React from 'react'
import { Pagination as RAPagination } from 'react-admin'

const Pagination = (props) => (
  <RAPagination rowsPerPageOptions={[15, 25, 50]} {...props} />
)

export default Pagination
