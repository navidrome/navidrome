import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'

// Search field names used by SearchInput across different list views:
// - 'name': AlbumList, ArtistList, LibraryList, PlayerList, RadioList, UserList
// - 'title': SongList
// - 'q': PlaylistList
// If a new list view uses a different source field, add it here.
const SEARCH_FIELDS = ['name', 'title', 'q']

const getSearchValue = (filter) => {
  for (const field of SEARCH_FIELDS) {
    if (filter[field]) return filter[field]
  }
  return ''
}

export const useSearchRefocus = () => {
  const location = useLocation()
  const prevSearchValue = useRef(null)

  useEffect(() => {
    const params = new URLSearchParams(location.search)
    const filterStr = params.get('filter') || '{}'

    let filter = {}
    try {
      filter = JSON.parse(filterStr)
    } catch (e) {
      // Invalid JSON, ignore
    }

    const searchValue = getSearchValue(filter)

    if (prevSearchValue.current && !searchValue) {
      // Use requestAnimationFrame to wait for React to finish re-rendering
      // after the URL change before focusing the input
      requestAnimationFrame(() => {
        // Selector depends on react-admin's internal class naming.
        // If react-admin changes these class names, this will need updating.
        const input = document.querySelector('[class*="RaSearchInput"] input')
        if (input) {
          input.focus()
        }
      })
    }

    prevSearchValue.current = searchValue
  }, [location.search])
}
