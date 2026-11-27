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
    if (record.mediaFileId) {
      // A playlistTrack id is a position, not a stable key: rating a song can drop it out
      // of a smart playlist, and that position then holds a different track. Refetching
      // the row by id would write the neighbour's data under this row, so reload the list.
      dataProvider
        .getOne('song', { id: record.mediaFileId })
        .catch((e) => {
          // eslint-disable-next-line no-console
          console.log('Error encountered: ' + e)
        })
        .finally(() => {
          refresh()
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
  }, [dataProvider, record.id, record.mediaFileId, refresh, resource])

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
