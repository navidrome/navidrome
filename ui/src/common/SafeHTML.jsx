import DOMPurify from 'dompurify'
import { useMemo } from 'react'

export const SafeHTML = ({ children }) => {
  const purified = useMemo(() => {
    const purify = DOMPurify()

    purify.addHook('afterSanitizeElements', async (node) => {
      if (node instanceof HTMLElement) {
        // Set referrer-policy for elements with src
        switch (node.tagName.toLowerCase()) {
          case 'a':
          case 'area':
          case 'img':
          case 'video':
          case 'iframe':
          case 'script':
            node.setAttribute('referrer-policy', 'no-referrer')
        }
      }
    })

    return purify.sanitize(children, { ADD_ATTR: ['referrer-policy'] })
  }, [children])

  return <span dangerouslySetInnerHTML={{ __html: purified }} />
}
