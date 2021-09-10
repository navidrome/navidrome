import React from 'react'
import { MenuItemLink, useGetList } from 'react-admin'
import QueueMusicOutlinedIcon from '@material-ui/icons/QueueMusicOutlined'
import SubMenu from './SubMenu'

const PlaylistsSubMenu = ({ open, sidebarIsOpen, dense, handleToggle }) => {
  const { data, ids } = useGetList(
    'playlist',
    {
      page: 0,
      perPage: 0,
    },
    { order: 'name' }
  )

  const renderPlaylistMenuItemLink = (pls) => {
    return (
      <MenuItemLink
        key={pls.id}
        to={`/playlist/${pls.id}/show`}
        primaryText={pls.name}
        sidebarIsOpen={sidebarIsOpen}
        dense={false}
      />
    )
  }

  return (
    <SubMenu
      handleToggle={handleToggle}
      isOpen={open}
      sidebarIsOpen={sidebarIsOpen}
      name={'menu.playlist'}
      icon={<QueueMusicOutlinedIcon />}
      dense={dense}
    >
      {ids.map((id) => renderPlaylistMenuItemLink(data[id]))}
    </SubMenu>
  )
}

export default PlaylistsSubMenu
