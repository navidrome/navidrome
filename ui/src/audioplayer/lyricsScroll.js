import {
  KARAOKE_SCROLL_ANIMATION_MS,
  KARAOKE_SCROLL_SETTLE_PX,
  clamp,
  easeInOut,
} from './lyricsKaraokeConstants'

export const cancelScrollAnimation = (scrollAnimationRef) => {
  const animation = scrollAnimationRef.current
  if (animation?.frameId != null) {
    window.cancelAnimationFrame(animation.frameId)
  }
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
  const maxTop = Math.max(0, body.scrollHeight - body.clientHeight)
  const nextTargetTop = clamp(targetTop, 0, maxTop)
  const distance = nextTargetTop - startTop
  if (Math.abs(distance) < KARAOKE_SCROLL_SETTLE_PX) return

  if (reducedMotion) {
    body.scrollTop = nextTargetTop
    return
  }

  const startedAt = performance.now()
  const animation = { frameId: null }
  const step = () => {
    const progress = clamp(
      (performance.now() - startedAt) / KARAOKE_SCROLL_ANIMATION_MS,
      0,
      1,
    )
    body.scrollTop = startTop + distance * easeInOut(progress)

    if (progress < 1) {
      animation.frameId = window.requestAnimationFrame(step)
      return
    }

    if (scrollAnimationRef.current === animation) {
      scrollAnimationRef.current = null
    }
  }

  animation.frameId = window.requestAnimationFrame(step)
  scrollAnimationRef.current = animation
}
