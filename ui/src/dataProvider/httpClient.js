import { fetchUtils } from 'react-admin'
import { v4 as uuidv4 } from 'uuid'
import { baseUrl } from '../utils'
import config from '../config'
import { jwtDecode } from 'jwt-decode'
import { removeHomeCache } from '../utils/removeHomeCache'

export const customAuthorizationHeader = 'X-ND-Authorization'
export const clientUniqueIdHeader = 'X-ND-Client-Unique-Id'
export const clientUniqueId = uuidv4()

const applyAuthHeaders = (headers) => {
  headers.set(clientUniqueIdHeader, clientUniqueId)
  const token = localStorage.getItem('token')
  if (token) {
    headers.set(customAuthorizationHeader, `Bearer ${token}`)
  }
  return headers
}

// Endpoints that return 204 No Content (e.g. radio now-playing) must not use fetchJson.
export const httpVoid = (url, options = {}) => {
  url = baseUrl(url)
  const headers = applyAuthHeaders(
    options.headers instanceof Headers
      ? options.headers
      : new Headers({ Accept: 'application/json', ...options.headers }),
  )
  return fetch(url, { ...options, headers }).then((response) => {
    const refreshedToken = response.headers.get(customAuthorizationHeader)
    if (refreshedToken) {
      const decoded = jwtDecode(refreshedToken)
      localStorage.setItem('token', refreshedToken)
      localStorage.setItem('userId', decoded.uid)
      config.firstTime = false
      removeHomeCache()
    }
    if (!response.ok) {
      throw new Error(`Request failed with status ${response.status}`)
    }
    return response
  })
}

const httpClient = (url, options = {}) => {
  url = baseUrl(url)
  if (!options.headers) {
    options.headers = new Headers({ Accept: 'application/json' })
  }
  applyAuthHeaders(options.headers)
  return fetchUtils.fetchJson(url, options).then((response) => {
    const refreshedToken = response.headers.get(customAuthorizationHeader)
    if (refreshedToken) {
      const decoded = jwtDecode(refreshedToken)
      localStorage.setItem('token', refreshedToken)
      localStorage.setItem('userId', decoded.uid)
      // Avoid going to create admin dialog after logout/login without a refresh
      config.firstTime = false
      removeHomeCache()
    }
    return response
  })
}

export default httpClient
