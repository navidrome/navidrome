import React, { useEffect, useRef } from 'react'
import { Pagination as RAPagination, useListContext } from 'react-admin'
import { abortAllInFlight } from './useImageUrl'

export const Pagination = (props) => {
  const { page, perPage } = useListContext()
  const prevRef = useRef({ page, perPage })

  useEffect(() => {
    const prev = prevRef.current
    if (prev.page !== page || prev.perPage !== perPage) {
      abortAllInFlight()
      prevRef.current = { page, perPage }
    }
  }, [page, perPage])

  return <RAPagination rowsPerPageOptions={[15, 25, 50]} {...props} />
}
