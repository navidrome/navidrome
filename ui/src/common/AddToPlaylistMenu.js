import React from 'react'
import { useDataProvider, useGetList, useNotify } from 'react-admin'
import { MenuItem } from '@material-ui/core'
import PropTypes from 'prop-types'

export const addTracksToPlaylist = (dataProvider, selectedIds, playlistId) =>
  dataProvider
    .create('playlistTrack', {
      data: { ids: selectedIds },
      filter: { playlist_id: playlistId },
    })
    .then(() => selectedIds.length)

export const addAlbumToPlaylist = (dataProvider, albumId, playlistId) =>
  dataProvider
    .getList('albumSong', {
      pagination: { page: 1, perPage: -1 },
      sort: { field: 'discNumber asc, trackNumber asc', order: 'ASC' },
      filter: { album_id: albumId },
    })
    .then((response) => response.data.map((song) => song.id))
    .then((ids) => addTracksToPlaylist(dataProvider, ids, playlistId))

const AddToPlaylistMenu = React.forwardRef(
  ({ selectedIds, albumId, onClose, onItemAdded }, ref) => {
    const notify = useNotify()
    const dataProvider = useDataProvider()
    const { ids, data, loaded } = useGetList(
      'playlist',
      { page: 1, perPage: -1 },
      { field: 'name', order: 'ASC' },
      {}
    )

    if (!loaded) {
      return <MenuItem>Loading...</MenuItem>
    }

    const handleItemClick = (e) => {
      e.preventDefault()
      const playlistId = e.target.getAttribute('value')
      if (playlistId !== '') {
        const add = albumId
          ? addAlbumToPlaylist(dataProvider, albumId, playlistId)
          : addTracksToPlaylist(dataProvider, selectedIds, playlistId)

        add
          .then((len) => {
            notify('message.songsAddedToPlaylist', 'info', { smart_count: len })
            onItemAdded && onItemAdded(playlistId, selectedIds)
          })
          .catch(() => {
            notify('ra.page.error', 'warning')
          })
      }
      e.stopPropagation()
      onClose && onClose(e)
    }

    return (
      <>
        {ids.map((id) => (
          <MenuItem value={id} key={id} onClick={handleItemClick}>
            {data[id].name}
          </MenuItem>
        ))}
      </>
    )
  }
)

AddToPlaylistMenu.propTypes = {
  selectedIds: PropTypes.arrayOf(PropTypes.any).isRequired,
  albumId: PropTypes.string,
  onClose: PropTypes.func,
  onItemAdded: PropTypes.func,
}

AddToPlaylistMenu.defaultProps = {
  selectedIds: [],
}

export default AddToPlaylistMenu
