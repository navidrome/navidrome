import { createLocation, createPath } from 'history'

// A drop-in replacement for history v4's createHashHistory that keeps the
// `#/path` URLs but changes them with pushState/replaceState instead of
// assigning location.hash / calling location.replace().
//
// Safari treats those fragment navigations as full page loads on insecure
// origins (plain HTTP on anything other than localhost), so with the stock hash
// history every route change, including each debounced keystroke in a list's
// search field, reloaded the whole app. History API updates are same-document
// in every browser. See https://github.com/navidrome/navidrome/issues/6363

const stripHash = (url) => {
  const i = url.indexOf('#')
  return i === -1 ? url : url.slice(0, i)
}

// Read from href rather than location.hash, which some browsers decode
const getHashPath = () => {
  const href = window.location.href
  const i = href.indexOf('#')
  return i === -1 ? '' : href.substring(i + 1)
}

const addLeadingSlash = (path) => (path.charAt(0) === '/' ? path : '/' + path)

const createKey = () => Math.random().toString(36).substring(2, 8)

const defaultGetUserConfirmation = (message, callback) =>
  callback(window.confirm(message))

export const createHashHistory = ({
  getUserConfirmation = defaultGetUserConfirmation,
} = {}) => {
  let listeners = []
  let prompt = null
  let ignoreNextPop = false

  // Our entries carry { key, state, idx }. idx lets a blocked back/forward be
  // undone; entries created outside this history (e.g. editing the address
  // bar) have no idx.
  const readEntry = () => window.history.state || {}

  const readLocation = () => {
    const { key, state } = readEntry()
    return createLocation(addLeadingSlash(getHashPath()), state, key)
  }

  const createHref = (location) => '#' + addLeadingSlash(createPath(location))

  const writeEntry = (method, location, idx) =>
    window.history[method](
      { key: location.key, state: location.state, idx },
      '',
      stripHash(window.location.href) + createHref(location),
    )

  const history = {
    length: window.history.length,
    action: 'POP',
    location: readLocation(),
  }
  let currentIdx = readEntry().idx ?? 0

  // Normalize the URL (e.g. a bare /app/ becomes /app/#/) and tag the entry
  writeEntry('replaceState', history.location, currentIdx)

  const setState = (action, location) => {
    history.action = action
    history.location = location
    history.length = window.history.length
    listeners.forEach((listener) => listener(location, action))
  }

  const confirmTransition = (location, action, callback) => {
    if (prompt === null) return callback(true)
    const result =
      typeof prompt === 'function' ? prompt(location, action) : prompt
    if (typeof result === 'string') {
      getUserConfirmation(result, callback)
    } else {
      callback(result !== false)
    }
  }

  const navigate = (action, path, state) => {
    const location = createLocation(path, state, createKey(), history.location)
    confirmTransition(location, action, (ok) => {
      if (!ok) return
      const samePath = createPath(location) === createPath(history.location)
      // Like history v4, pushing the current path does not add an entry
      if (action === 'PUSH' && !samePath) {
        currentIdx += 1
        writeEntry('pushState', location, currentIdx)
      } else {
        writeEntry('replaceState', location, currentIdx)
      }
      setState(action, location)
    })
  }

  const handlePop = () => {
    const location = readLocation()
    // A fragment navigation fires both popstate and hashchange
    if (
      createPath(location) === createPath(history.location) &&
      location.key === history.location.key
    ) {
      return
    }
    if (ignoreNextPop) {
      ignoreNextPop = false
      return
    }
    const { idx } = readEntry()
    confirmTransition(location, 'POP', (ok) => {
      if (ok) {
        if (idx === undefined) {
          // Entry created outside this history: tag it so it can be revisited
          writeEntry('replaceState', location, ++currentIdx)
        } else {
          currentIdx = idx
        }
        setState('POP', location)
      } else if (idx !== undefined && idx !== currentIdx) {
        ignoreNextPop = true
        window.history.go(currentIdx - idx)
      } else {
        // Can't tell how far we moved; put the previous URL back in place
        writeEntry('replaceState', history.location, currentIdx)
      }
    })
  }

  window.addEventListener('popstate', handlePop)
  window.addEventListener('hashchange', handlePop)

  return Object.assign(history, {
    createHref,
    push: (path, state) => navigate('PUSH', path, state),
    replace: (path, state) => navigate('REPLACE', path, state),
    go: (n) => window.history.go(n),
    goBack: () => window.history.go(-1),
    goForward: () => window.history.go(1),
    block: (newPrompt = false) => {
      prompt = newPrompt
      return () => {
        if (prompt === newPrompt) prompt = null
      }
    },
    listen: (listener) => {
      listeners = [...listeners, listener]
      return () => {
        listeners = listeners.filter((l) => l !== listener)
      }
    },
    destroy: () => {
      window.removeEventListener('popstate', handlePop)
      window.removeEventListener('hashchange', handlePop)
      listeners = []
    },
  })
}
