const ALBUM_MODE_GRID = 'ALBUM_GRID_MODE'
const ALBUM_MODE_LIST = 'ALBUM_LIST_MODE'
const selectViewMode = (mode) => ({ type: mode })

const ALBUM_LIST_ALL = 'ALBUM_LIST_ALL'
const ALBUM_LIST_RANDOM = 'ALBUM_LIST_RANDOM'
const ALBUM_LIST_NEWEST = 'ALBUM_LIST_NEWEST'
const ALBUM_LIST_RECENT = 'ALBUM_LIST_RECENT'
const ALBUM_LIST_STARRED = 'ALBUM_LIST_STARRED'

const albumListParams = {
  ALBUM_LIST_ALL: { sort: { field: 'name', order: 'ASC' } },
  ALBUM_LIST_RANDOM: { sort: { field: 'random' } },
  ALBUM_LIST_NEWEST: { sort: { field: 'created_at', order: 'DESC' } },
  ALBUM_LIST_RECENT: {
    sort: { field: 'play_date', order: 'DESC' },
    filter: { starred: true }
  }
}

const selectAlbumList = (mode) => ({ type: mode })

const albumViewReducer = (
  previousState = {
    mode: localStorage.getItem('albumViewMode') || ALBUM_MODE_LIST,
    list: localStorage.getItem('albumListType') || ALBUM_LIST_ALL,
    params: { sort: {}, filter: {} }
  },
  payload
) => {
  const { type } = payload
  switch (type) {
    case ALBUM_MODE_GRID:
    case ALBUM_MODE_LIST:
      localStorage.setItem('albumViewMode', type)
      return { ...previousState, mode: type }
    case ALBUM_LIST_ALL:
    case ALBUM_LIST_RANDOM:
    case ALBUM_LIST_NEWEST:
    case ALBUM_LIST_RECENT:
    case ALBUM_LIST_STARRED:
      localStorage.setItem('albumListType', type)
      return { ...previousState, list: type, params: albumListParams[type] }
    default:
      return previousState
  }
}

export {
  ALBUM_MODE_LIST,
  ALBUM_MODE_GRID,
  ALBUM_LIST_ALL,
  ALBUM_LIST_RANDOM,
  ALBUM_LIST_NEWEST,
  ALBUM_LIST_RECENT,
  ALBUM_LIST_STARRED,
  albumViewReducer,
  selectViewMode,
  selectAlbumList
}
