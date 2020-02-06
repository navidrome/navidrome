import jwtDecode from 'jwt-decode'

const authProvider = {
  login: ({ username, password }) => {
    let url = '/app/login'
    if (localStorage.getItem('initialAccountCreation')) {
      url = '/app/createAdmin'
    }
    const request = new Request(url, {
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
        localStorage.removeItem('initialAccountCreation')
        localStorage.setItem('token', response.token)
        localStorage.setItem('name', response.name)
        localStorage.setItem('username', response.username)
        localStorage.setItem('role', response.isAdmin ? 'admin' : 'regular')
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

  checkAuth: () =>
    localStorage.getItem('token') ? Promise.resolve() : Promise.reject(),

  checkError: (error) => {
    const { status, message } = error
    if (message === 'no users created') {
      localStorage.setItem('initialAccountCreation', 'true')
    }
    if (status === 401 || status === 403) {
      removeItems()
      return Promise.reject()
    }
    return Promise.resolve()
  },

  getPermissions: () => {
    const role = localStorage.getItem('role')
    return role ? Promise.resolve(role) : Promise.reject()
  }
}

const removeItems = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('name')
  localStorage.removeItem('username')
  localStorage.removeItem('role')
}

export default authProvider
