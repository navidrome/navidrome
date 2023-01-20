import { useCallback, useMemo, useState } from 'react'

// Idea from https://blog.bitsrc.io/new-react-design-pattern-return-component-from-hooks-79215c3eac00
export const useDialog = () => {
  const [anchorEl, setAnchorEl] = useState(null)

  const open = useCallback((event) => {
    event?.stopPropagation()
    setAnchorEl(event.currentTarget)
  }, [])

  const close = useCallback((event) => {
    event?.stopPropagation()
    setAnchorEl(null)
  }, [])

  const props = useMemo(() => {
    return {
      anchorEl,
      open: Boolean(anchorEl),
      onClose: close,
    }
  }, [anchorEl, close])

  return {
    open,
    close,
    props,
  }
}
