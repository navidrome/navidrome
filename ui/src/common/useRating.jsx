import { useState, useCallback, useEffect, useRef } from 'react'
import { useDataProvider, useNotify } from 'react-admin'
import subsonic from '../subsonic'

export const useRating = (resource, record) => {
  const [loading, setLoading] = useState(false)
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const mountedRef = useRef(false)
  const rating = record.rating

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const refreshRating = useCallback(() => {
    dataProvider
      .getOne(resource, { id: record.id })
      .then(() => {
        if (mountedRef.current) {
          setLoading(false)
        }
      })
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.log('Error encountered: ' + e)
      })
  }, [dataProvider, record, resource])

  const rate = (val, id) => {
    setLoading(true)
    subsonic
      .setRating(id, val)
      .then(refreshRating)
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.log('Error setting star rating: ', e)
        notify('ra.page.error', 'warning')
        if (mountedRef.current) {
          setLoading(false)
        }
      })
  }

  return [rate, rating, loading]
}
