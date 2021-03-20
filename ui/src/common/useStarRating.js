import { useState, useCallback, useEffect, useRef } from 'react'
import { useDataProvider, useNotify } from 'react-admin'
import subsonic from '../subsonic'

export const useStarRating = (resource, record = {}) => {
  const [hover, setHover] = useState(null)
  const [loading, setLoading] = useState(false)
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const mountedRef = useRef(false)

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
        console.log('Error encountered: ' + e)
      })
  }, [dataProvider, record.id, resource])

  const rate = (val) => {
    setLoading(true)
    subsonic
      .setRating(record.id, val)
      .then(refreshRating)
      .catch((e) => {
        console.log('Error setting star rating: ', e)
        notify('ra.page.error', 'warning')
        if (mountedRef.current) {
          setLoading(false)
        }
      })
  }

  const hoverRating = useCallback(
    (val) => {
      setHover(val)
    },
    [hover]
  )

  return [rate, hoverRating, hover, loading]
}
