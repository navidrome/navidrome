import { useState, useCallback, useEffect, useRef } from 'react'
import { useDataProvider, useNotify, useRefresh } from 'react-admin'
import subsonic from '../subsonic'

export const useRating = (resource, record) => {
  const [loading, setLoading] = useState(false)
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const refresh = useRefresh()
  const mountedRef = useRef(false)
  const rating = record.rating

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const refreshRating = useCallback(() => {
    // Playlist track IDs are positional, so refetching the old ID after a
    // filter change can cache the next track under the wrong row key.
    if (record.mediaFileId) {
      const promises = [dataProvider.getOne('song', { id: record.mediaFileId })]

      Promise.all(promises)
        .then(() => refresh())
        .catch((e) => {
          // eslint-disable-next-line no-console
          console.log('Error encountered: ' + e)
        })
        .finally(() => {
          if (mountedRef.current) {
            setLoading(false)
          }
        })
    } else {
      // Regular song or other resource
      dataProvider
        .getOne(resource, { id: record.id })
        .catch((e) => {
          // eslint-disable-next-line no-console
          console.log('Error encountered: ' + e)
        })
        .finally(() => {
          if (mountedRef.current) {
            setLoading(false)
          }
        })
    }
  }, [
    dataProvider,
    record.id,
    record.mediaFileId,
    refresh,
    resource,
  ])

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
