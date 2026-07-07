import { KARAOKE_SCROLL_SETTLE_PX, clamp } from './lyricsKaraokeConstants'

export const cancelScrollAnimation = (scrollAnimationRef) => {
  const animation = scrollAnimationRef.current
  if (animation?.frameId) window.cancelAnimationFrame(animation.frameId)
  scrollAnimationRef.current = null
}

export const getScrollEndPadding = (body, anchorRatio) => {
  const height = Number(body?.clientHeight)
  if (!Number.isFinite(height) || height <= 0) return 0
  return Math.round(height * (1 - anchorRatio))
}

export const getAnchoredScrollTop = (body, targetNode, anchorRatio) => {
  const bodyRect = body.getBoundingClientRect()
  const targetRect = targetNode.getBoundingClientRect()
  const activeAnchorTop = body.clientHeight * anchorRatio
  const deltaWithinBody = targetRect.top - bodyRect.top - activeAnchorTop
  const maxTop = Math.max(0, body.scrollHeight - body.clientHeight)
  return clamp(body.scrollTop + deltaWithinBody, 0, maxTop)
}

export const animateScrollTop = ({
  body,
  targetTop,
  reducedMotion,
  scrollAnimationRef,
}) => {
  cancelScrollAnimation(scrollAnimationRef)

  const startTop = body.scrollTop
  const distance = targetTop - startTop
  if (Math.abs(distance) < KARAOKE_SCROLL_SETTLE_PX) return

  const maxTop = Math.max(0, body.scrollHeight - body.clientHeight)
  const nextTargetTop = clamp(targetTop, 0, maxTop)
  if (reducedMotion) {
    body.scrollTop = nextTargetTop
    return
  }

  body.scrollTo({ top: nextTargetTop, behavior: 'smooth' })
}
