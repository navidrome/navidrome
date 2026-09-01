import { useCallback, useEffect, useRef, useState } from 'react'
import { useDataProvider, useNotify, useRefresh } from 'react-admin'
import subsonic from '../subsonic'

export const useToggleLove = (resource, record = {}) => {
  const [loading, setLoading] = useState(false)
  const notify = useNotify()
  const refresh = useRefresh()

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

    // Playlist track IDs are positional, so refetching the old ID after a
    // filter change can cache the next track under the wrong row key.
    if (!record.playlistId) {
      promises.push(dataProvider.getOne(resource, { id: record.id }))
    }

    // If we have a mediaFileId, also refresh the song
    if (record.mediaFileId) {
      promises.push(dataProvider.getOne('song', { id: record.mediaFileId }))
    }

    Promise.all(promises)
      .then(() => {
        if (record.playlistId) {
          refresh()
        }
      })
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.log('Error encountered: ' + e)
      })
      .finally(() => {
        if (mountedRef.current) {
          setLoading(false)
        }
      })
  }, [
    dataProvider,
    record.mediaFileId,
    record.id,
    record.playlistId,
    refresh,
    resource,
  ])

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
