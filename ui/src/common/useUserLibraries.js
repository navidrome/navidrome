import { useCallback, useEffect } from 'react'
import { useDispatch } from 'react-redux'
import { useDataProvider } from 'react-admin'
import { setUserLibraries } from '../actions'
import { useRefreshOnEvents } from './useRefreshOnEvents'

/**
 * Loads the current user's accessible libraries into the Redux store and keeps
 * them refreshed when library/user events occur. Mount this once high in the
 * tree (e.g. the Layout) so consumers like useSelectedLibraries always have the
 * data available, regardless of whether the sidebar/LibrarySelector is open.
 */
export const useUserLibraries = () => {
  const dispatch = useDispatch()
  const dataProvider = useDataProvider()

  const loadUserLibraries = useCallback(async () => {
    const userId = localStorage.getItem('userId')
    if (!userId) return
    try {
      const { data } = await dataProvider.getOne('user', { id: userId })
      dispatch(setUserLibraries(data.libraries || []))
    } catch (error) {
      // eslint-disable-next-line no-console
      console.warn(
        'Could not load user libraries (this may be expected for non-admin users):',
        error,
      )
    }
  }, [dataProvider, dispatch])

  useEffect(() => {
    loadUserLibraries()
  }, [loadUserLibraries])

  useRefreshOnEvents({
    events: ['library', 'user'],
    onRefresh: loadUserLibraries,
  })
}
