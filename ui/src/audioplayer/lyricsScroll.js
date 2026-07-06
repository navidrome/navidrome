import {
  KARAOKE_SCROLL_DURATION_MS,
  KARAOKE_SCROLL_SETTLE_PX,
  clamp,
  easeOutCubic,
} from './lyricsKaraokeConstants'

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

  if (reducedMotion) {
    body.scrollTop = targetTop
    return
  }

  const startTime = performance.now()

  const animation = { frameId: 0 }
  const step = (now) => {
    if (scrollAnimationRef.current !== animation) return

    const elapsed = now - startTime
    const progress = Math.min(elapsed / KARAOKE_SCROLL_DURATION_MS, 1)
    const maxTop = Math.max(0, body.scrollHeight - body.clientHeight)
    const nextTargetTop = clamp(targetTop, 0, maxTop)
    body.scrollTop =
      startTop + (nextTargetTop - startTop) * easeOutCubic(progress)

    if (
      progress < 1 &&
      Math.abs(body.scrollTop - nextTargetTop) >= KARAOKE_SCROLL_SETTLE_PX
    ) {
      animation.frameId = window.requestAnimationFrame(step)
      return
    }

    body.scrollTop = nextTargetTop
    scrollAnimationRef.current = null
  }

  scrollAnimationRef.current = animation
  animation.frameId = window.requestAnimationFrame(step)
}
