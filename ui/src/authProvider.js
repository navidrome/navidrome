import jwtDecode from 'jwt-decode'
import md5 from 'blueimp-md5'
import { v4 as uuidv4 } from 'uuid'
import { baseUrl } from './utils'
import config from './config'
import { startEventStream, stopEventStream } from './eventStream'

if (
  config.auth &&
  /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(config.auth.id)
) {
  localStorage.setItem('token', config.auth.token)
  localStorage.setItem('userId', config.auth.id)
  localStorage.setItem('name', config.auth.name)
  localStorage.setItem('username', config.auth.username)
  config.auth.avatar && config.auth.setItem('avatar', config.auth.avatar)
  localStorage.setItem('role', config.auth.isAdmin ? 'admin' : 'regular')
  localStorage.setItem('subsonic-salt', config.auth.subsonicSalt)
  localStorage.setItem('subsonic-token', config.auth.subsonicToken)
}

const authProvider = {
  login: ({ username, password }) => {
    let url = baseUrl('/app/login')
    if (config.firstTime) {
      url = baseUrl('/app/createAdmin')
    }
    const request = new Request(url, {
      method: 'POST',
      body: JSON.stringify({ username, password }),
      headers: new Headers({ 'Content-Type': 'application/json' }),
    })
    return fetch(request)
      .then((response) => {
        if (response.status < 200 || response.status >= 300) {
          throw new Error(response.statusText)
        }
        return response.json()
      })
      .then((response) => {
        // Validate token
        jwtDecode(response.token)
        // TODO Store all items in one object
        localStorage.setItem('token', response.token)
        localStorage.setItem('userId', response.id)
        localStorage.setItem('name', response.name)
        localStorage.setItem('username', response.username)
        response.avatar && localStorage.setItem('avatar', response.avatar)
        localStorage.setItem('role', response.isAdmin ? 'admin' : 'regular')
        const salt = generateSubsonicSalt()
        localStorage.setItem('subsonic-salt', salt)
        localStorage.setItem(
          'subsonic-token',
          generateSubsonicToken(password, salt)
        )
        // Avoid going to create admin dialog after logout/login without a refresh
        config.firstTime = false
        if (config.devActivityPanel) {
          startEventStream()
        }
        return response
      })
      .catch((error) => {
        if (
          error.message === 'Failed to fetch' ||
          error.stack === 'TypeError: Failed to fetch'
        ) {
          throw new Error('errors.network_error')
        }

        throw new Error(error)
      })
  },

  logout: () => {
    stopEventStream()
    removeItems()
    try {
      clearServiceWorkerCache()
    } catch (e) {
      console.log('Error clearing service worker cache:', e)
    }
    return Promise.resolve()
  },

  checkAuth: () =>
    localStorage.getItem('token') ? Promise.resolve() : Promise.reject(),

  checkError: ({ status }) => {
    if (status === 401) {
      removeItems()
      return Promise.reject()
    }
    return Promise.resolve()
  },

  getPermissions: () => {
    const role = localStorage.getItem('role')
    return role ? Promise.resolve(role) : Promise.reject()
  },

  getIdentity: () => {
    return {
      id: localStorage.getItem('username'),
      fullName: localStorage.getItem('name'),
      avatar: localStorage.getItem('avatar'),
    }
  },
}

const removeItems = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('userId')
  localStorage.removeItem('name')
  localStorage.removeItem('username')
  localStorage.removeItem('avatar')
  localStorage.removeItem('role')
  localStorage.removeItem('subsonic-salt')
  localStorage.removeItem('subsonic-token')
}

const clearServiceWorkerCache = () => {
  window.caches &&
    caches.keys().then(function (keyList) {
      for (let key of keyList) caches.delete(key)
    })
}

const generateSubsonicSalt = () => {
  const h = md5(uuidv4())
  return h.slice(0, 6)
}

const generateSubsonicToken = (password, salt) => {
  return md5(password + salt)
}

export default authProvider
