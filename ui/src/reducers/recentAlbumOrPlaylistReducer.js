import { RECENT_ALBUM, RECENT_PLAYLIST, RECENT_RESET } from '../actions'

const initialState = {
    type: '',
    name: ''
}

export const recentAlbumOrPlaylistReducer = (previousState = initialState, payload) => {
    const { type } = payload
    switch (type) {
        case RECENT_ALBUM:
            console.log('RECENT Album Reducer')
            return previousState
        case RECENT_PLAYLIST:
            console.log('RECENT Playlist Reducer')
            return previousState
        case RECENT_RESET:
            console.log("RECENT Reset")
            return previousState
        default:
            return previousState
    }
}
