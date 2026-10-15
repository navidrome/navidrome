import { useSelector } from 'react-redux'
import { useState } from 'react'
import { useRefresh, useDataProvider } from 'react-admin'

/**
 * A hook that automatically refreshes react-admin managed resources when refresh events are received via SSE.
 *
 * This hook is designed for components that display react-admin managed resources (like lists, shows, edits)
 * and need to stay in sync when those resources are modified elsewhere in the application.
 *
 * **When to use this hook:**
 * - Your component displays react-admin resources (albums, songs, artists, playlists, etc.)
 * - You want automatic refresh when those resources are created/updated/deleted
 * - Your data comes from standard dataProvider.getMany() calls
 * - You're using react-admin's data management (queries, mutations, caching)
 *
 * **When NOT to use this hook:**
 * - Your component displays derived/custom data not directly managed by react-admin
 * - You need custom reload logic beyond dataProvider.getMany()
 * - Your data comes from non-standard endpoints
 * - Use `useRefreshOnEvents` instead for these scenarios
 *
 * @param {...string} visibleResources - Resource names to watch for changes.
 *   If no resources specified, watches all resources.
 *   If '*' is included in resources, triggers full page refresh.
 *
 * @example
 * // Example 1: Album list - refresh when albums change
 * const AlbumList = () => {
 *   useResourceRefresh('album')
 *   return <List resource="album">...</List>
 * }
 *
 * @example
 * // Example 2: Album show page - refresh when album or its songs change
 * const AlbumShow = () => {
 *   useResourceRefresh('album', 'song')
 *   return <Show resource="album">...</Show>
 * }
 *
 * @example
 * // Example 3: Dashboard - refresh when any resource changes
 * const Dashboard = () => {
 *   useResourceRefresh() // No parameters = watch all resources
 *   return <div>...</div>
 * }
 *
 * @example
 * // Example 4: Library management page - watch library resources
 * const LibraryList = () => {
 *   useResourceRefresh('library')
 *   return <List resource="library">...</List>
 * }
 *
 * **How it works:**
 * - Listens to refresh events from the SSE connection
 * - When events arrive, checks if they match the specified visible resources
 * - For specific resource IDs: calls dataProvider.getMany(resource, {ids: [...]})
 * - For global refreshes: calls refresh() to reload the entire page
 * - Uses react-admin's built-in data management and caching
 *
 * **Event format expected:**
 * - Global refresh: { '*': '*' } or { someResource: ['*'] }
 * - Specific resources: { album: ['id1', 'id2'], song: ['id3'] }
 */
// Resources whose records are media files, and so inherit their album's artwork.
const trackResources = ['song', 'playlistTrack']

export const useResourceRefresh = (...visibleResources) => {
  const [lastTime, setLastTime] = useState(Date.now())
  const refresh = useRefresh()
  const dataProvider = useDataProvider()
  const refreshData = useSelector(
    (state) => state.activity?.refresh || { lastReceived: lastTime },
  )
  const loadedResources = useSelector((state) => state.admin?.resources)
  const { resources, lastReceived } = refreshData

  if (lastReceived <= lastTime) {
    return
  }
  setLastTime(lastReceived)

  const isWatched = (r) =>
    visibleResources.length === 0 || visibleResources.includes(r)
  // A wildcard on a resource this component does not show is somebody else's business: reloading
  // the page for it throws away the list the user is looking at.
  const hasWildcard =
    resources &&
    (resources['*'] === '*' ||
      Object.entries(resources).some(
        ([r, ids]) => isWatched(r) && ids.includes?.('*'),
      ))

  if (hasWildcard) {
    refresh()
    return
  }
  if (!resources) {
    return
  }
  Object.keys(resources).forEach((r) => {
    if (isWatched(r)) {
      if (resources[r]?.length > 0) {
        // Only refetch records already in the store; ones the UI never loaded will
        // arrive fresh (with the new artwork) when navigated to, so fetching them is wasteful.
        const loaded = loadedResources?.[r]?.data || {}
        const ids = resources[r].filter((id) => loaded[id] !== undefined)
        if (ids.length > 0) {
          dataProvider.getMany(r, { ids })
        }
      }
    }
  })

  // A track with no art of its own is served its album's, so an album's new coverArt id moves its
  // tracks' too. The dependent ids are unbounded server-side, but the store knows which are loaded.
  if (resources.album?.length > 0) {
    const albumIds = new Set(resources.album)
    trackResources.filter(isWatched).forEach((r) => {
      const ids = Object.values(loadedResources?.[r]?.data || {})
        .filter((t) => albumIds.has(t?.albumId))
        .map((t) => t.id)
      if (ids.length > 0) {
        dataProvider.getMany(r, { ids })
      }
    })
  }
}
