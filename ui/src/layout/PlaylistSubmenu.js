import React, { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import { useRefresh, MenuItemLink, useGetList } from 'react-admin'
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
  const refresh = useRefresh()
  const [playLists, setPlaylists] = useState([])
  const { data } = useGetList(
    'playlist',
    { page: 1, perPage: -1 },
    { field: 'name', order: 'ASC' },
    {}
  )

  useEffect(() => {
    if (data && typeof data === 'object') {
      const isEmpty = !Object.keys(data).length
      if (!isEmpty) {
        setPlaylists(Object.values(data))
      } else if (isEmpty && playLists.length) {
        refresh()
      }
    }
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
          playLists.map(({ id, name }) => (
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
