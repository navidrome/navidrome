import React from 'react'
import { Pagination as RAPagination } from 'react-admin'

const perPage = [15, 25, 50]
var perPageCustom = parseInt(localStorage.rowsPerPageOther)

if (perPageCustom > 0) {
  perPage.push(perPageCustom)
  perPage.sort(function (a, b) {
    return a - b
  })
}

const Pagination = (props) => (
  <RAPagination rowsPerPageOptions={perPage} {...props} />
)

export default Pagination
