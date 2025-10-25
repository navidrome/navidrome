import jsonServerProvider from 'ra-data-json-server'
import httpClient from './httpClient'
import { REST_URL } from '../consts'

const dataProvider = jsonServerProvider(REST_URL, httpClient)

const isAdmin = () => {
  const role = localStorage.getItem('role')
  return role === 'admin'
}

const getSelectedLibraries = () => {
  try {
    const state = JSON.parse(localStorage.getItem('state'))
    return state?.library?.selectedLibraries || []
  } catch (err) {
    return []
  }
}

// Function to apply library filtering to appropriate resources
const applyLibraryFilter = (resource, params) => {
  // Content resources that should be filtered by selected libraries
  const filteredResources = ['album', 'song', 'artist', 'playlistTrack', 'tag']

  // Get selected libraries from localStorage
  const selectedLibraries = getSelectedLibraries()

  // Add library filter for content resources if libraries are selected
  if (filteredResources.includes(resource) && selectedLibraries.length > 0) {
    if (!params.filter) {
      params.filter = {}
    }
    params.filter.library_id = selectedLibraries
  }

  return params
}

const mapResource = (resource, params) => {
  switch (resource) {
    // /api/playlistTrack?playlist_id=123  => /api/playlist/123/tracks
    case 'playlistTrack': {
      params.filter = params.filter || {}

      let plsId = '0'
      plsId = params.filter.playlist_id
      if (!isAdmin()) {
        params.filter.missing = false
      }
      params = applyLibraryFilter(resource, params)

      return [`playlist/${plsId}/tracks`, params]
    }
    case 'album':
    case 'song':
    case 'artist':
    case 'tag': {
      params.filter = params.filter || {}
      if (!isAdmin()) {
        params.filter.missing = false
      }
      params = applyLibraryFilter(resource, params)

      return [resource, params]
    }
    default:
      return [resource, params]
  }
}

const callDeleteMany = (resource, params) => {
  const ids = (params.ids || []).map((id) => `id=${id}`)
  const query = ids.length > 0 ? `?${ids.join('&')}` : ''
  return httpClient(`${REST_URL}/${resource}${query}`, {
    method: 'DELETE',
  }).then((response) => ({ data: response.json.ids || [] }))
}

// Helper function to handle user-library associations
const handleUserLibraryAssociation = async (userId, libraryIds) => {
  if (!libraryIds || libraryIds.length === 0) {
    return // Admin users or users without library assignments
  }

  try {
    await httpClient(`${REST_URL}/user/${userId}/library`, {
      method: 'PUT',
      body: JSON.stringify({ libraryIds }),
    })
  } catch (error) {
    console.error('Error setting user libraries:', error) //eslint-disable-line no-console
    throw error
  }
}

// Enhanced user creation that handles library associations
const createUser = async (params) => {
  const { data } = params
  const { libraryIds, ...userData } = data

  // First create the user
  const userResponse = await dataProvider.create('user', { data: userData })
  const userId = userResponse.data.id

  // Then set library associations for non-admin users
  if (!userData.isAdmin && libraryIds && libraryIds.length > 0) {
    await handleUserLibraryAssociation(userId, libraryIds)
  }

  return userResponse
}

// Enhanced user update that handles library associations
const updateUser = async (params) => {
  const { data } = params
  const { libraryIds, ...userData } = data
  const userId = params.id

  // First update the user
  const userResponse = await dataProvider.update('user', {
    ...params,
    data: userData,
  })

  // Then handle library associations for non-admin users
  if (!userData.isAdmin && libraryIds !== undefined) {
    await handleUserLibraryAssociation(userId, libraryIds)
  }

  return userResponse
}

const wrapperDataProvider = {
  ...dataProvider,
  getList: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.getList(r, p)
  },
  getOne: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    const response = dataProvider.getOne(r, p)

    // Transform user data to ensure libraryIds is present for form compatibility
    if (resource === 'user') {
      return response.then((result) => {
        if (result.data.libraries && Array.isArray(result.data.libraries)) {
          result.data.libraryIds = result.data.libraries.map((lib) => lib.id)
        }
        return result
      })
    }

    return response
  },
  getMany: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.getMany(r, p)
  },
  getManyReference: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.getManyReference(r, p)
  },
  update: (resource, params) => {
    if (resource === 'user') {
      return updateUser(params)
    }
    const [r, p] = mapResource(resource, params)
    return dataProvider.update(r, p)
  },
  updateMany: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.updateMany(r, p)
  },
  create: (resource, params) => {
    if (resource === 'user') {
      return createUser(params)
    }
    const [r, p] = mapResource(resource, params)
    return dataProvider.create(r, p)
  },
  delete: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.delete(r, p)
  },
  deleteMany: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    if (r.endsWith('/tracks') || resource === 'missing') {
      return callDeleteMany(r, p)
    }
    return dataProvider.deleteMany(r, p)
  },
  addToPlaylist: (playlistId, data) => {
    return httpClient(`${REST_URL}/playlist/${playlistId}/tracks`, {
      method: 'POST',
      body: JSON.stringify(data),
    }).then(({ json }) => ({ data: json }))
  },
  getPlaylists: (songId) => {
    return httpClient(`${REST_URL}/song/${songId}/playlists`).then(
      ({ json }) => ({ data: json }),
    )
  },
  inspect: (songId) => {
    return httpClient(`${REST_URL}/inspect?id=${songId}`).then(({ json }) => ({
      data: json,
    }))
  },
}

export default wrapperDataProvider
