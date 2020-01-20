import jwtDecode from 'jwt-decode'

const authProvider = {
  login: ({ username, password }) => {
    const request = new Request('/app/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
      headers: new Headers({ 'Content-Type': 'application/json' })
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
        localStorage.setItem('token', response.token)
        localStorage.setItem('name', response.name)
        localStorage.setItem('username', response.username)
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
    removeItems()
    return Promise.resolve()
  },

  checkAuth: () => {
    try {
      const expireTime = jwtDecode(localStorage.getItem('token')).exp * 1000
      const now = new Date().getTime()
      return now < expireTime ? Promise.resolve() : Promise.reject()
    } catch (e) {
      return Promise.reject()
    }
  },

  checkError: (error) => {
    const { status } = error
    // TODO Remove 403?
    if (status === 401 || status === 403) {
      removeItems()
      return Promise.reject()
    }
    return Promise.resolve()
  },

  getPermissions: (params) => Promise.resolve()
}

const removeItems = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('name')
  localStorage.removeItem('username')
}

export default authProvider
