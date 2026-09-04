import { useCallback, useEffect, useRef, useState } from 'react'
import { useDataProvider, useNotify, useRefresh } from 'react-admin'
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
  const refresh = useRefresh()

  const refreshRecord = useCallback(() => {
    // A playlistTrack id is a position, not a stable key: loving a song can drop it out of
    // a smart playlist, and that position then holds a different track. Refetching the row
    // by id would write the neighbour's data under this row, so reload the list instead.
    const isPlaylistTrack = !!record.mediaFileId
    const target = isPlaylistTrack
      ? { resource: 'song', params: { id: record.mediaFileId } }
      : { resource, params: { id: record.id } }

    dataProvider
      .getOne(target.resource, target.params)
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.log('Error encountered: ' + e)
      })
      .finally(() => {
        if (isPlaylistTrack) {
          refresh()
        }
        if (mountedRef.current) {
          setLoading(false)
        }
      })
  }, [dataProvider, record.mediaFileId, record.id, refresh, resource])

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
