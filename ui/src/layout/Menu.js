import React, { useState, createElement, useEffect } from 'react'
import { useSelector } from 'react-redux'
import { useMediaQuery } from '@material-ui/core'
import {
  useTranslate,
  MenuItemLink,
  getResources,
  useGetList,
  useRefresh,
} from 'react-admin'
import { withRouter } from 'react-router-dom'
import { makeStyles } from '@material-ui/core/styles'
import LibraryMusicIcon from '@material-ui/icons/LibraryMusic'
import ViewListIcon from '@material-ui/icons/ViewList'
import AlbumIcon from '@material-ui/icons/Album'
import SubMenu from './SubMenu'
import inflection from 'inflection'
import albumLists from '../album/albumLists'
import { HelpDialog } from '../dialogs'
import Playlist from '../icons/Playlist'

const translatedResourceName = (resource, translate) =>
  translate(`resources.${resource.name}.name`, {
    smart_count: 2,
    _:
      resource.options && resource.options.label
        ? translate(resource.options.label, {
            smart_count: 2,
            _: resource.options.label,
          })
        : inflection.humanize(inflection.pluralize(resource.name)),
  })

const useStyles = makeStyles({
  menuWrapper: {
    maxHeight: '100%',
    display: 'flex',
    flexDirection: 'column',
  },
})

const Menu = ({ onMenuClick, dense, logout }) => {
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

  const subItems = (subMenu) => (resource) => {
    const { hasList, options, name } = resource
    return hasList && name !== 'playlist' && options?.subMenu === subMenu
  }

  const RenderPlaylistLinks = ({ playlist }) => {
    const { id, name } = playlist
    return (
      <MenuItemLink
        key={id}
        to={`/playlist/${id}/show`}
        primaryText={name}
        onClick={onMenuClick}
        sidebarIsOpen={open}
        dense={dense}
      />
    )
  }

  const classes = useStyles()
  const refresh = useRefresh()
  const [playLists, setPlaylists] = useState([])
  const { data } = useGetList('playlist', { page: 1, perPage: -1 }, {}, {})

  useEffect(() => {
    const isEmpty = !Object.keys(data).length

    if (!isEmpty) {
      setPlaylists(Object.values(data))
    } else if (isEmpty && playLists.length) {
      refresh()
    }
    // eslint-disable-next-line
  }, [data])

  return (
    <div className={classes.menuWrapper}>
      <div>
        <SubMenu
          handleToggle={() => handleToggle('menuAlbumList')}
          isOpen={state.menuAlbumList}
          sidebarIsOpen={open}
          name="menu.albumList"
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
          name="menu.library"
          icon={<LibraryMusicIcon />}
          dense={dense}
        >
          {resources
            .filter(subItems('library'))
            .map(renderResourceMenuItemLink)}
        </SubMenu>
      </div>
      <div
        style={{
          overflowY: open && state.menuPlaylists ? 'scroll' : 'auto',
          overflowX: 'hidden',
        }}
      >
        <SubMenu
          handleToggle={() => handleToggle('menuPlaylists')}
          isOpen={state.menuPlaylists}
          sidebarIsOpen={open}
          name="resources.playlist.name"
          icon={<Playlist />}
          dense={dense}
          secondaryAction={onMenuClick}
          secondaryLink="/playlist"
        >
          {open &&
            playLists.map((playlist) => (
              <RenderPlaylistLinks key={playlist.id} playlist={playlist} />
            ))}
        </SubMenu>
      </div>
      {resources.filter(subItems(undefined)).map(renderResourceMenuItemLink)}
      {isXsmall && logout}
      <HelpDialog />
    </div>
  )
}

export default withRouter(Menu)
