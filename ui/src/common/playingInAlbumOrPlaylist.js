import PropTypes from 'prop-types'

//songId is the ID of the playlist or album corresponding to the song
//activeId is the ID of the current album or playlist
export const playingInAlbumOrPlaylist = (current, activeId, songId) => {
    if (Object.keys(current).length !== 0 && !current.paused) {
        if (activeId === songId) {
            return true
        }
    }
    return false
}

playingInAlbumOrPlaylist.propTypes = {
    current: PropTypes.object.isRequired,
    activeId: PropTypes.string,
    songId: PropTypes.string
}