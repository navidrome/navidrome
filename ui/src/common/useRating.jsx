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
    // For playlist tracks, refresh both resources to keep data in sync
    if (record.mediaFileId) {
      // This is a playlist track - refresh both the playlist track and the song
      const promises = [
        dataProvider.getOne('song', { id: record.mediaFileId }),
        dataProvider.getOne('playlistTrack', {
          id: record.id,
          filter: { playlist_id: record.playlistId },
        }),
      ]

      Promise.all(promises)
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
  }, [dataProvider, record.id, record.mediaFileId, record.playlistId, resource])

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
