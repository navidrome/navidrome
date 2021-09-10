import React from 'react'
import { MenuItemLink, useGetList, useTranslate } from 'react-admin'
import Playlist from '../icons/Playlist'
import SubMenu from './SubMenu'

const PlaylistsSubMenu = ({
  open,
  sidebarIsOpen,
  dense,
  handleToggle,
  onMenuClick,
}) => {
  const translate = useTranslate()
  const name = translate('resources.playlist.name', { smart_count: 2 })
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
        onClick={onMenuClick}
        // leftIcon={<QueueMusicIcon />}
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
      name={name}
      icon={<Playlist />}
      dense={dense}
    >
      {ids.map((id) => renderPlaylistMenuItemLink(data[id]))}
    </SubMenu>
  )
}

export default PlaylistsSubMenu
