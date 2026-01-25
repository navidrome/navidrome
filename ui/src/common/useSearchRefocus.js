import { useEffect } from 'react'

export const useSearchRefocus = () => {
  useEffect(() => {
    const handleClick = (e) => {
      const clearButton = e.target.closest('[aria-label*="clear" i]')
      if (clearButton) {
        setTimeout(() => {
          const searchInput = document.querySelector('[class*="RaSearchInput"] input')
          if (searchInput) {
            searchInput.focus()
          }
        }, 800)
      }
    }
    document.addEventListener('click', handleClick, true)
    return () => document.removeEventListener('click', handleClick, true)
  }, [])
}
