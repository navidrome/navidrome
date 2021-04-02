import React, { useState, createElement } from 'react'
import { useSelector } from 'react-redux'
import { useMediaQuery } from '@material-ui/core'
import { useTranslate, MenuItemLink, getResources } from 'react-admin'
import { withRouter } from 'react-router-dom'
import { makeStyles } from '@material-ui/core/styles'
import LibraryMusicIcon from '@material-ui/icons/LibraryMusic'
import ViewListIcon from '@material-ui/icons/ViewList'
import AlbumIcon from '@material-ui/icons/Album'
import SubMenu from './SubMenu'
import albumLists from '../album/albumLists'
import { HelpDialog } from '../dialogs'
import PlaylistSubmenu from './PlaylistSubmenu'
import { translatedResourceName } from '../utils'

const useStyles = makeStyles({
  menuWrapper: {
    maxHeight: '100%',
    display: 'flex',
    flexDirection: 'column',
  },
})

const Menu = ({ onMenuClick, dense, logout }) => {
  console.log(dense)
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const open = useSelector((state) => state.admin.ui.sidebarOpen)
  const translate = useTranslate()
  const resources = useSelector(getResources)
  // TODO State is not persisted in mobile when you close the sidebar menu. Move to redux?
  const [state, setState] = useState({
    menuAlbumList: true,
    menuLibrary: true,
    menuPlaylists: true,
    menuSettings: false,
  })

  const handleToggle = (menu) => {
    setState((state) => ({ ...state, [menu]: !state[menu] }))
  }

  const renderResourceMenuItemLink = (resource) => (
    <MenuItemLink
      key={resource.name}
      to={`/${resource.name}`}
      primaryText={translatedResourceName(resource, translate)}
      leftIcon={
        (resource.icon && createElement(resource.icon)) || <ViewListIcon />
      }
      onClick={onMenuClick}
      sidebarIsOpen={open}
      dense={dense}
    />
  )

  const renderAlbumMenuItemLink = (type, al) => {
    const resource = resources.find((r) => r.name === 'album')
    if (!resource) {
      return null
    }

    const albumListAddress = `/album/${type}`

    const name = translate(`resources.album.lists.${type || 'default'}`, {
      _: translatedResourceName(resource, translate),
    })

    return (
      <MenuItemLink
        key={albumListAddress}
        to={albumListAddress}
        primaryText={name}
        leftIcon={(al.icon && createElement(al.icon)) || <ViewListIcon />}
        onClick={onMenuClick}
        sidebarIsOpen={open}
        dense={dense}
        exact
      />
    )
  }

  const subItems = (subMenu) => ({ hasList, options }) =>
    hasList && options && options.subMenu === subMenu

  const classes = useStyles()

  return (
    <div className={classes.menuWrapper}>
      <div>
        <SubMenu
          handleToggle={() => handleToggle('menuAlbumList')}
          isOpen={state.menuAlbumList}
          sidebarIsOpen={open}
          name={translate('menu.albumList')}
          icon={<AlbumIcon />}
          dense={dense}
        >
          {Object.keys(albumLists).map((type) =>
            renderAlbumMenuItemLink(type, albumLists[type])
          )}
        </SubMenu>
        <SubMenu
          handleToggle={() => handleToggle('menuLibrary')}
          isOpen={state.menuLibrary}
          sidebarIsOpen={open}
          name={translate('menu.library')}
          icon={<LibraryMusicIcon />}
          dense={dense}
        >
          {resources
            .filter(subItems('library'))
            .map(renderResourceMenuItemLink)}
        </SubMenu>
      </div>
      <PlaylistSubmenu
        handleToggle={() => handleToggle('menuPlaylists')}
        isToggled={state.menuPlaylists}
        isSidebarOpen={open}
        dense={dense}
        onMenuClick={onMenuClick}
        resources={resources}
      />
      {resources.filter(subItems(undefined)).map(renderResourceMenuItemLink)}
      {isXsmall && logout}
      <HelpDialog />
    </div>
  )
}

export default withRouter(Menu)
