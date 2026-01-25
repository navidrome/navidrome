import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'

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

    const searchValue = filter.name || filter.title || filter.q || ''

    if (prevSearchValue.current && !searchValue) {
      setTimeout(() => {
        const input = document.querySelector('[class*="RaSearchInput"] input')
        if (input) {
          input.focus()
        }
      }, 100)
    }

    prevSearchValue.current = searchValue
  }, [location.search])
}
