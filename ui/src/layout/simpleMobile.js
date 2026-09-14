// Simple mobile mode: opt-in only.
// Default is FULL UI. Enable with localStorage nd.simpleMobile=1 or the Simple mode menu.
// Never blank the app: any error falls back to full UI.

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

// Opt-in only. Auto-enabling on phone UA caused a blank screen after login for some users.
export const shouldUseSimpleMobile = () => {
  try {
    return readStoredPref() === '1'
  } catch (e) {
    return false
  }
}

export const setSimpleMobilePref = (on) => {
  try {
    localStorage.setItem(SIMPLE_MOBILE_KEY, on ? '1' : '0')
  } catch (e) {
    // ignore
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

export const simpleMobilePlayerProps = () => {
  if (!shouldUseSimpleMobile()) {
    return {}
  }
  return { responsive: false, toggleMode: false }
}
