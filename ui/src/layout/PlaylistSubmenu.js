import React, { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import {
  MenuItemLink,
  useGetList,
  useDataProvider,
  useRefresh,
} from 'react-admin'
import Playlist from '../icons/Playlist'
import SubMenu from './SubMenu'

const PlaylistSubmenu = ({
  isSidebarOpen,
  isToggled,
  name,
  dense,
  handleToggle,
  onMenuClick,
}) => {
  const [playlists, setPlaylists] = useState([])
  const { data } = useGetList('playlist', {}, {})
  const dataProvider = useDataProvider()
  const refresh = useRefresh()

  const setPlaylistData = () => {
    dataProvider
      .getList('playlist', {
        pagination: { page: 1, perPage: -1 },
        sort: { field: 'name', order: 'ASC' },
      })
      .then((res) => {
        if (res?.data) setPlaylists(Object.values(res.data))
      })
  }
  useEffect(() => {
    setPlaylistData()
    refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data])

  return (
    <div style={{ overflow: 'auto' }}>
      <SubMenu
        handleToggle={() => handleToggle('menuPlaylists')}
        isOpen={isToggled}
        sidebarIsOpen={isSidebarOpen}
        name={name}
        icon={<Playlist />}
        dense={dense}
        secondaryAction={onMenuClick}
        secondaryLink="/playlist"
      >
        {isSidebarOpen &&
          playlists.map(({ id, name }) => (
            <MenuItemLink
              key={id}
              to={`/playlist/${id}/show`}
              primaryText={name}
              onClick={onMenuClick}
              sidebarIsOpen={isSidebarOpen}
              dense={dense}
            />
          ))}
      </SubMenu>
    </div>
  )
}

export default PlaylistSubmenu

PlaylistSubmenu.propTypes = {
  isSidebarOpen: PropTypes.bool.isRequired,
  isToggled: PropTypes.bool.isRequired,
  name: PropTypes.string.isRequired,
  handleToggle: PropTypes.func.isRequired,
  onMenuClick: PropTypes.func.isRequired,
  dense: PropTypes.bool,
}
