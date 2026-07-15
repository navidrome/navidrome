import { useCallback, useEffect, useRef, useState } from 'react'
import { useDataProvider, useNotify } from 'react-admin'
import { useDispatch } from 'react-redux'
import subsonic from '../subsonic'
import { updateQueueSkipped } from '../actions'

export const useToggleSkip = (resource, record = {}) => {
  const [loading, setLoading] = useState(false)
  const notify = useNotify()
  const dispatch = useDispatch()

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

  const toggleSkip = () => {
    const nextSkipped = !record.skipped
    const toggle = record.skipped ? subsonic.unskip : subsonic.skip
    const id = record.mediaFileId || record.id

    setLoading(true)
    toggle(id)
      .then(() => {
        dispatch(updateQueueSkipped(id, nextSkipped))
        refreshRecord()
      })
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.log('Error toggling skip: ', e)
        notify('ra.page.error', 'warning')
        if (mountedRef.current) {
          setLoading(false)
        }
      })
  }

  return [toggleSkip, loading]
}
