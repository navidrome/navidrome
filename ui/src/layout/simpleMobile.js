// Simple mobile mode: a two-button shell instead of the full React-Admin chrome.
//
// Detection (evaluated synchronously so the first React paint can skip the full UI):
//   1. localStorage `nd.simpleMobile` is "1" or "0" and wins if set
//   2. otherwise treat as mobile when the UA looks like a phone OR the viewport is
//      ≤600px (Navidrome/MUI `xs`) AND the pointer is coarse (touch)
// Default on an unset key: simple mode for those mobile devices, full UI elsewhere.

export const SIMPLE_MOBILE_KEY = 'nd.simpleMobile'

const PHONE_UA =
  /Android.+Mobile|iPhone|iPod|webOS|BlackBerry|IEMobile|Opera Mini/i

const match = (query) => {
  if (typeof window === 'undefined' || !window.matchMedia) {
    return false
  }
  return window.matchMedia(query).matches
}

export const isMobileDevice = () => {
  if (typeof navigator === 'undefined') {
    return false
  }
  const phoneUA = PHONE_UA.test(navigator.userAgent)
  const narrow = match('(max-width: 600px)')
  const coarse = match('(pointer: coarse)')
  return phoneUA || (narrow && coarse)
}

export const shouldUseSimpleMobile = () => {
  if (typeof localStorage === 'undefined') {
    return false
  }
  const stored = localStorage.getItem(SIMPLE_MOBILE_KEY)
  if (stored === '0') {
    return false
  }
  if (stored === '1') {
    return true
  }
  return isMobileDevice()
}

export const setSimpleMobilePref = (on) => {
  localStorage.setItem(SIMPLE_MOBILE_KEY, on ? '1' : '0')
  window.location.reload()
}

export const applySimpleMobileDomHint = () => {
  if (typeof document === 'undefined') {
    return
  }
  if (shouldUseSimpleMobile()) {
    document.documentElement.setAttribute('data-simple-mobile', '1')
  } else {
    document.documentElement.removeAttribute('data-simple-mobile')
  }
}
