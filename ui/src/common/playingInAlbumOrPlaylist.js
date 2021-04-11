import PropTypes from 'prop-types'

/**
 * Description of the function
 * @param {string} songID - the ID of the playlist or album corresponding to the song
 * @param {string} activeID - the ID of the current album or playlist - the ID of the playlist or album corresponding to the song
 * @returns {boolean}
 */

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
  songId: PropTypes.string,
}
