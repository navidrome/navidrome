import { useState, useEffect, useCallback } from 'react'

/**
 * Manages image loading/error state and lightbox open/close.
 * Resets when recordId changes.
 */
export const useImageLoadingState = (recordId) => {
  const [imageLoading, setImageLoading] = useState(true)
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
