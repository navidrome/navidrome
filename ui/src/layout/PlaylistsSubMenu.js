import React, { useCallback } from 'react'
import { MenuItemLink, useQueryWithStore } from 'react-admin'
import { useHistory } from 'react-router-dom'
import QueueMusicIcon from '@material-ui/icons/QueueMusic'
import { Typography } from '@material-ui/core'
import QueueMusicOutlinedIcon from '@material-ui/icons/QueueMusicOutlined'
import { BiCog } from 'react-icons/all'
import SubMenu from './SubMenu'

const PlaylistsSubMenu = ({ state, setState, sidebarIsOpen, dense }) => {
  const history = useHistory()
  const { data, loaded } = useQueryWithStore({
    type: 'getList',
    resource: 'playlist',
    payload: {
      pagination: {
        page: 0,
        perPage: 0,
      },
      sort: { field: 'name' },
    },
  })

  const handleToggle = (menu) => {
    setState((state) => ({ ...state, [menu]: !state[menu] }))
  }

  const renderPlaylistMenuItemLink = (pls) => {
    return (
      <MenuItemLink
        key={pls.id}
        to={`/playlist/${pls.id}/show`}
        primaryText={
          <Typography variant="inherit" noWrap>
            {pls.name}
          </Typography>
        }
        sidebarIsOpen={sidebarIsOpen}
        dense={false}
      />
    )
  }

  const user = localStorage.getItem('username')
  const myPlaylists = []
  const sharedPlaylists = []

  if (loaded) {
    const allPlaylists = Object.keys(data).map((id) => data[id])

    allPlaylists.forEach((pls) => {
      if (user === pls.owner) {
        myPlaylists.push(pls)
      } else {
        sharedPlaylists.push(pls)
      }
    })
  }

  const onPlaylistConfig = useCallback(
    () => history.push('/playlist'),
    [history]
  )

  return (
    <>
      <SubMenu
        handleToggle={() => handleToggle('menuPlaylists')}
        isOpen={state.menuPlaylists}
        sidebarIsOpen={sidebarIsOpen}
        name={'menu.playlists'}
        icon={<QueueMusicIcon />}
        dense={dense}
        actionIcon={<BiCog />}
        onAction={onPlaylistConfig}
      >
        {myPlaylists.map(renderPlaylistMenuItemLink)}
      </SubMenu>
      {sharedPlaylists?.length > 0 && (
        <SubMenu
          handleToggle={() => handleToggle('menuSharedPlaylists')}
          isOpen={state.menuSharedPlaylists}
          sidebarIsOpen={sidebarIsOpen}
          name={'menu.sharedPlaylists'}
          icon={<QueueMusicOutlinedIcon />}
          dense={dense}
        >
          {sharedPlaylists.map(renderPlaylistMenuItemLink)}
        </SubMenu>
      )}
    </>
  )
}

export default PlaylistsSubMenu
