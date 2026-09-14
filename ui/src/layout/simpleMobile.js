// Simple mobile mode: a two-button shell instead of the full React-Admin chrome.
//
// Detection (evaluated synchronously so the first React paint can skip the full UI):
//   1. localStorage `nd.simpleMobile` is "1" or "0" and wins if set
//   2. otherwise treat as mobile when the UA looks like a phone OR the viewport is
//      ≤600px (Navidrome/MUI `xs`) AND the pointer is coarse (touch)
// Default on an unset key: simple mode for those mobile devices, full UI elsewhere.
// On any storage/UA error: fall back to FULL UI (never blank the app).

export const SIMPLE_MOBILE_KEY = 'nd.simpleMobile'

const PHONE_UA =
  /Android.+Mobile|iPhone|iPod|webOS|BlackBerry|IEMobile|Opera Mini/i

const match = (query) => {
  if (typeof window === 'undefined' || !window.matchMedia) {
    return false
  }
  try {
    return window.matchMedia(query).matches
  } catch (e) {
    return false
  }
}

const readStoredPref = () => {
  try {
    if (typeof localStorage === 'undefined') {
      return null
    }
    return localStorage.getItem(SIMPLE_MOBILE_KEY)
  } catch (e) {
    return null
  }
}

export const isMobileDevice = () => {
  try {
    if (typeof navigator === 'undefined') {
      return false
    }
    const phoneUA = PHONE_UA.test(navigator.userAgent || '')
    const narrow = match('(max-width: 600px)')
    const coarse = match('(pointer: coarse)')
    return phoneUA || (narrow && coarse)
  } catch (e) {
    return false
  }
}

export const shouldUseSimpleMobile = () => {
  try {
    const stored = readStoredPref()
    if (stored === '0') {
      return false
    }
    if (stored === '1') {
      return true
    }
    return isMobileDevice()
  } catch (e) {
    return false
  }
}

export const setSimpleMobilePref = (on) => {
  try {
    localStorage.setItem(SIMPLE_MOBILE_KEY, on ? '1' : '0')
  } catch (e) {
    // ignore quota / private mode
  }
  window.location.reload()
}

export const applySimpleMobileDomHint = () => {
  if (typeof document === 'undefined') {
    return
  }
  try {
    if (shouldUseSimpleMobile()) {
      document.documentElement.setAttribute('data-simple-mobile', '1')
    } else {
      document.documentElement.removeAttribute('data-simple-mobile')
    }
  } catch (e) {
    document.documentElement.removeAttribute('data-simple-mobile')
  }
}

// jinke's mobile `mode: 'full'` is a fullscreen overlay. Use the desktop bottom
// bar instead so the two simple-mode buttons stay tappable.
export const simpleMobilePlayerProps = () => {
  if (!shouldUseSimpleMobile()) {
    return {}
  }
  return { responsive: false, toggleMode: false }
}
