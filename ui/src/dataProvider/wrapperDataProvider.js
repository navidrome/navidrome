import jsonServerProvider from 'ra-data-json-server'
import httpClient from './httpClient'

const restUrl = '/app/api'

const dataProvider = jsonServerProvider(restUrl, httpClient)

const mapResource = (resource, params) => {
  console.log('R: ', resource, 'P: ', params)
  switch (resource) {
    case 'albumSong':
      return ['song', params]

    case 'playlistTrack':
      // /api/playlistTrack?playlist_id=123  => /api/playlist/123/tracks
      let plsId = '0'
      if (params.filter) {
        plsId = params.filter.playlist_id
        delete params.filter.playlist_id
      }
      return [`playlist/${plsId}/tracks`, params]

    default:
      return [resource, params]
  }
}

const wrapperDataProvider = {
  ...dataProvider,
  getList: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.getList(r, p)
  },
  getOne: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.getOne(r, p)
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
    const [r, p] = mapResource(resource, params)
    return dataProvider.update(r, p)
  },
  updateMany: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.updateMany(r, p)
  },
  create: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.create(r, p)
  },
  delete: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.delete(r, p)
  },
  deleteMany: (resource, params) => {
    const [r, p] = mapResource(resource, params)
    return dataProvider.deleteMany(r, p)
  },
}

export default wrapperDataProvider
