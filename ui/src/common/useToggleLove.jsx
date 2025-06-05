import { useCallback, useEffect, useRef, useState } from 'react'
import { useDataProvider, useNotify } from 'react-admin'
import subsonic from '../subsonic'

export const useToggleLove = (resource, record = {}) => {
  const [loading, setLoading] = useState(false)
  const notify = useNotify()

  const mountedRef = useRef(false)
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const dataProvider = useDataProvider()

  const refreshRecord = useCallback(() => {
    const promises = []

    // Always refresh the original resource
    const params = { id: record.id }
    if (record.playlistId) {
      params.filter = { playlist_id: record.playlistId }
    }
    promises.push(dataProvider.getOne(resource, params))

    // If we have a mediaFileId, also refresh the song
    if (record.mediaFileId) {
      promises.push(dataProvider.getOne('song', { id: record.mediaFileId }))
    }

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
  }, [dataProvider, record.mediaFileId, record.id, record.playlistId, resource])

  const toggleLove = () => {
    const toggle = record.starred ? subsonic.unstar : subsonic.star
    const id = record.mediaFileId || record.id

    setLoading(true)
    toggle(id)
      .then(refreshRecord)
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.log('Error toggling love: ', e)
        notify('ra.page.error', 'warning')
        if (mountedRef.current) {
          setLoading(false)
        }
      })
  }

  return [toggleLove, loading]
}
