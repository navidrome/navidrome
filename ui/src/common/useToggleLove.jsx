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
    dataProvider.getOne(resource, { id: record.id }).then(() => {
      if (mountedRef.current) {
        setLoading(false)
      }
    })
  }, [dataProvider, record.id, resource])

  const toggleLove = () => {
    const toggle = record.starred ? subsonic.unstar : subsonic.star

    setLoading(true)
    toggle(record.id)
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
