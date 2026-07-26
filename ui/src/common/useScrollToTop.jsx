import { useEffect } from 'react'

// React Router keeps the previous page's scroll offset, so a detail page opened from a scrolled
// list starts mid-page. Keyed on the record id so detail-to-detail navigation resets too.
export const useScrollToTop = (key) => {
  useEffect(() => {
    if (key) {
      window.scrollTo({ top: 0 })
    }
  }, [key])
}
