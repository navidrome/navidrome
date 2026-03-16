import { useState, useEffect, useCallback } from 'react'

/**
 * Manages image loading/error state and lightbox open/close for artist detail views.
 * Resets when record.id changes.
 */
const useArtistImageState = (recordId) => {
  const [imageLoading, setImageLoading] = useState(false)
  const [imageError, setImageError] = useState(false)
  const [isLightboxOpen, setLightboxOpen] = useState(false)

  useEffect(() => {
    setImageLoading(true)
    setImageError(false)
  }, [recordId])

  const handleImageLoad = useCallback(() => {
    setImageLoading(false)
    setImageError(false)
  }, [])

  const handleImageError = useCallback(() => {
    setImageLoading(false)
    setImageError(true)
  }, [])

  const handleOpenLightbox = useCallback(() => {
    if (!imageError) {
      setLightboxOpen(true)
    }
  }, [imageError])

  const handleCloseLightbox = useCallback(() => setLightboxOpen(false), [])

  return {
    imageLoading,
    imageError,
    isLightboxOpen,
    handleImageLoad,
    handleImageError,
    handleOpenLightbox,
    handleCloseLightbox,
  }
}

export default useArtistImageState
