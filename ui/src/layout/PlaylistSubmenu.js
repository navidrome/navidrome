import React, { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import { useRefresh, MenuItemLink, useGetList, useTranslate } from 'react-admin'
import Playlist from '../icons/Playlist'
import SubMenu from './SubMenu'
import { translatedResourceName } from '../utils'

const PlaylistSubmenu = ({
  isSidebarOpen,
  isToggled,
  resources,
  dense,
  handleToggle,
  onMenuClick,
}) => {
  const refresh = useRefresh()
  const translate = useTranslate()
  const [playLists, setPlaylists] = useState([])
  const [menuName, setMenuName] = useState('')
  const { data } = useGetList('playlist', { page: 1, perPage: -1 }, {}, {})

  useEffect(() => {
    const isEmpty = !Object.keys(data).length
    if (!isEmpty) {
      setPlaylists(Object.values(data))
    } else if (isEmpty && playLists.length) {
      refresh()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data])

  useEffect(() => {
    if (resources?.length) {
      const playListResource = resources.find(
        (resource) => resource.name === 'playlist'
      )
      setMenuName(translatedResourceName(playListResource, translate))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resources])

  return (
    <div
      style={{
        overflowY: isSidebarOpen && isToggled ? 'scroll' : 'auto',
        overflowX: 'hidden',
      }}
    >
      <SubMenu
        handleToggle={() => handleToggle('menuPlaylists')}
        isOpen={isToggled}
        sidebarIsOpen={isSidebarOpen}
        name={menuName}
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
  resources: PropTypes.array.isRequired,
  handleToggle: PropTypes.func.isRequired,
  onMenuClick: PropTypes.func.isRequired,
  dense: PropTypes.bool,
}
