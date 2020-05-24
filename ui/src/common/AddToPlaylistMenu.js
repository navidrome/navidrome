import React from 'react'
import { useDispatch } from 'react-redux'
import PropTypes from 'prop-types'
import {
  useDataProvider,
  useGetList,
  useNotify,
  useTranslate,
} from 'react-admin'
import { MenuItem, Divider } from '@material-ui/core'
import NewPlaylistIcon from '@material-ui/icons/Add'
import { openNewPlaylist } from '../dialogs/dialogState'
import NewPlaylistDialog from '../dialogs/NewPlaylist'

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
    const dispatch = useDispatch()
    const translate = useTranslate()
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

    const addToPlaylist = (playlistId) => {
      const add = albumId
        ? addAlbumToPlaylist(dataProvider, albumId, playlistId)
        : addTracksToPlaylist(dataProvider, selectedIds, playlistId)

      add
        .then((len) => {
          notify('message.songsAddedToPlaylist', 'info', { smart_count: len })
          onItemAdded(playlistId)
        })
        .catch(() => {
          notify('ra.page.error', 'warning')
        })
    }

    const handleItemClick = (e) => {
      e.preventDefault()
      const playlistId = e.target.getAttribute('value')
      if (playlistId !== '') {
        addToPlaylist(playlistId)
      }
      e.stopPropagation()
      onClose(e)
    }

    const handleOpenDialog = (e) => {
      e.preventDefault()
      dispatch(openNewPlaylist(albumId, selectedIds))
      e.stopPropagation()
      onClose(e)
    }

    return (
      <>
        <Divider component="li" />
        {ids.map((id) => (
          <MenuItem value={id} key={id} onClick={handleItemClick}>
            {data[id].name}
          </MenuItem>
        ))}
        <MenuItem
          value="newPlaylist"
          key="newPlaylist"
          onClick={handleOpenDialog}
        >
          {<NewPlaylistIcon fontSize="small" />}&nbsp;
          {translate('resources.playlist.actions.newPlaylist')}
        </MenuItem>
        <NewPlaylistDialog onSubmit={onItemAdded} />
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
  onClose: () => {},
  onItemAdded: () => {},
}

export default AddToPlaylistMenu
