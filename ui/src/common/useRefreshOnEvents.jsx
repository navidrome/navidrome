import { useEffect, useState } from 'react'
import { useSelector } from 'react-redux'

/**
 * A reusable hook for triggering custom reload logic when specific SSE events occur.
 *
 * This hook is ideal when:
 * - Your component displays derived/related data that isn't directly managed by react-admin
 * - You need custom loading logic that goes beyond simple dataProvider.getMany() calls
 * - Your data comes from non-standard endpoints or requires special processing
 * - You want to reload parent/related resources when child resources change
 *
 * @param {Object} options - Configuration options
 * @param {Array<string>} options.events - Array of event types to listen for (e.g., ['library', 'user', '*'])
 * @param {Function} options.onRefresh - Async function to call when events occur.
 *   Should be wrapped in useCallback with appropriate dependencies to avoid unnecessary re-renders.
 *
 * @example
 * // Example 1: LibrarySelector - Reload user data when library changes
 * const loadUserLibraries = useCallback(async () => {
 *   const userId = localStorage.getItem('userId')
 *   if (userId) {
 *     const { data } = await dataProvider.getOne('user', { id: userId })
 *     dispatch(setUserLibraries(data.libraries || []))
 *   }
 * }, [dataProvider, dispatch])
 *
 * useRefreshOnEvents({
 *   events: ['library', 'user'],
 *   onRefresh: loadUserLibraries
 * })
 *
 * @example
 * // Example 2: Statistics Dashboard - Reload stats when any music data changes
 * const loadStats = useCallback(async () => {
 *   const stats = await dataProvider.customRequest('GET', 'stats')
 *   setDashboardStats(stats)
 * }, [dataProvider, setDashboardStats])
 *
 * useRefreshOnEvents({
 *   events: ['album', 'song', 'artist'],
 *   onRefresh: loadStats
 * })
 *
 * @example
 * // Example 3: Permission-based UI - Reload permissions when user changes
 * const loadPermissions = useCallback(async () => {
 *   const authData = await authProvider.getPermissions()
 *   setUserPermissions(authData)
 * }, [authProvider, setUserPermissions])
 *
 * useRefreshOnEvents({
 *   events: ['user'],
 *   onRefresh: loadPermissions
 * })
 *
 * @example
 * // Example 4: Listen to all events (use sparingly)
 * const reloadAll = useCallback(async () => {
 *   // This will trigger on ANY refresh event
 *   await reloadEverything()
 * }, [reloadEverything])
 *
 * useRefreshOnEvents({
 *   events: ['*'],
 *   onRefresh: reloadAll
 * })
 */
export const useRefreshOnEvents = ({ events, onRefresh }) => {
  const [lastRefreshTime, setLastRefreshTime] = useState(Date.now())

  const refreshData = useSelector(
    (state) => state.activity?.refresh || { lastReceived: lastRefreshTime },
  )

  useEffect(() => {
    const { resources, lastReceived } = refreshData

    // Only process if we have new events
    if (lastReceived <= lastRefreshTime) {
      return
    }

    // Check if any of the events we're interested in occurred
    const shouldRefresh =
      resources &&
      // Global refresh event
      (resources['*'] === '*' ||
        // Check for specific events we're listening to
        events.some((eventType) => {
          if (eventType === '*') {
            return true // Listen to all events
          }
          return resources[eventType] // Check if this specific event occurred
        }))

    if (shouldRefresh) {
      setLastRefreshTime(lastReceived)

      // Call the custom refresh function
      if (onRefresh) {
        onRefresh().catch((error) => {
          // eslint-disable-next-line no-console
          console.warn('Error in useRefreshOnEvents onRefresh callback:', error)
        })
      }
    }
  }, [refreshData, lastRefreshTime, events, onRefresh])
}
