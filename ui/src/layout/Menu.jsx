import React, { useState } from 'react'
import { useSelector } from 'react-redux'
import { Divider, makeStyles } from '@material-ui/core'
import clsx from 'clsx'
import { useTranslate, MenuItemLink, getResources } from 'react-admin'
import ViewListIcon from '@material-ui/icons/ViewList'
import AlbumIcon from '@material-ui/icons/Album'
import FolderIcon from '@material-ui/icons/Folder'
import CategoryIcon from '@material-ui/icons/Category'
import MoodIcon from '@material-ui/icons/Mood'
import LocalOfferIcon from '@material-ui/icons/LocalOffer'
import SubMenu from './SubMenu'
import { humanize, pluralize } from 'inflection'
import albumLists from '../album/albumLists'
import PlaylistsSubMenu from './PlaylistsSubMenu'
import LibrarySelector from '../common/LibrarySelector'
import config from '../config'
import { TAG_DASHBOARDS } from '../tagDashboard/tagDashboards'
import { getFeaturePermissions } from '../authProvider'

const useStyles = makeStyles((theme) => ({
  root: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
    transition: theme.transitions.create('width', {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.leavingScreen,
    }),
    paddingBottom: (props) => (props.addPadding ? '80px' : '20px'),
  },
  open: {
    width: 240,
  },
  closed: {
    width: 55,
  },
  active: {
    color: theme.palette.text.primary,
    fontWeight: 'bold',
  },
}))

const translatedResourceName = (resource, translate) =>
  translate(`resources.${resource.name}.name`, {
    smart_count: 2,
    _:
      resource.options && resource.options.label
        ? translate(resource.options.label, {
            smart_count: 2,
            _: resource.options.label,
          })
        : humanize(pluralize(resource.name)),
  })

const Menu = ({ dense = false }) => {
  const open = useSelector((state) => state.admin.ui.sidebarOpen)
  const translate = useTranslate()
  const queue = useSelector((state) => state.player?.queue)
  const classes = useStyles({ addPadding: queue.length > 0 })
  const resources = useSelector(getResources)
  // Personal show/hide toggles - always read unconditionally (React Hooks can't be called
  // conditionally). Admin-gated fork feature access is combined in afterward, below.
  const personalShowFolderView = useSelector(
    (state) => state.settings.showFolderView !== false,
  )
  const showPodcasts = useSelector(
    (state) => state.settings.showPodcasts !== false,
  )
  const showGenreView = useSelector(
    (state) => state.settings.showGenreView !== false,
  )
  const personalShowAiGenreView = useSelector(
    (state) => state.settings.showAiGenreView !== false,
  )
  const personalShowAiMoodView = useSelector(
    (state) => state.settings.showAiMoodView !== false,
  )
  const personalShowMyTagsView = useSelector(
    (state) => state.settings.showMyTagsView !== false,
  )

  // Admin-gated fork feature access (folders/ai_tags/my_tags) - a feature key absent from the
  // stored grant means enabled, matching the backend's opt-out default. Genre and Podcasts have no
  // admin gate yet (see model/user_feature_permission.go), so only their personal toggle applies.
  const featurePermissions = getFeaturePermissions()
  const showFolderView =
    featurePermissions.folders !== false && personalShowFolderView
  const showAiGenreView =
    featurePermissions.ai_tags !== false && personalShowAiGenreView
  const showAiMoodView =
    featurePermissions.ai_tags !== false && personalShowAiMoodView
  const showMyTagsView =
    featurePermissions.my_tags !== false && personalShowMyTagsView

  // TODO State is not persisted in mobile when you close the sidebar menu. Move to redux?
  const [state, setState] = useState({
    menuAlbumList: true,
    menuPlaylists: true,
    menuSharedPlaylists: true,
  })

  const handleToggle = (menu) => {
    setState((state) => ({ ...state, [menu]: !state[menu] }))
  }

  const renderResourceMenuItemLink = (resource) => (
    <MenuItemLink
      key={resource.name}
      to={`/${resource.name}`}
      activeClassName={classes.active}
      primaryText={translatedResourceName(resource, translate)}
      leftIcon={resource.icon || <ViewListIcon />}
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
        activeClassName={classes.active}
        primaryText={name}
        leftIcon={al.icon || <ViewListIcon />}
        sidebarIsOpen={open}
        dense={dense}
        exact
      />
    )
  }

  const subItems = (subMenu) => (resource) =>
    resource.hasList && resource.options && resource.options.subMenu === subMenu

  return (
    <div
      className={clsx(classes.root, {
        [classes.open]: open,
        [classes.closed]: !open,
      })}
    >
      {open && <LibrarySelector />}
      <SubMenu
        handleToggle={() => handleToggle('menuAlbumList')}
        isOpen={state.menuAlbumList}
        sidebarIsOpen={open}
        name="menu.albumList"
        icon={<AlbumIcon />}
        dense={dense}
      >
        {Object.keys(albumLists).map((type) =>
          renderAlbumMenuItemLink(type, albumLists[type]),
        )}
      </SubMenu>
      {showFolderView && (
        <MenuItemLink
          to="/folder"
          activeClassName={classes.active}
          primaryText={translate('menu.folders')}
          leftIcon={<FolderIcon />}
          sidebarIsOpen={open}
          dense={dense}
        />
      )}
      {showGenreView && (
        <MenuItemLink
          to="/genre"
          activeClassName={classes.active}
          primaryText={translate('resources.genre.name', { smart_count: 2 })}
          leftIcon={<CategoryIcon />}
          sidebarIsOpen={open}
          dense={dense}
        />
      )}
      {showAiGenreView && (
        <MenuItemLink
          to={TAG_DASHBOARDS.aiGenre.path}
          activeClassName={classes.active}
          primaryText={translate('resources.aiGenre.name', {
            smart_count: 2,
          })}
          leftIcon={<CategoryIcon />}
          sidebarIsOpen={open}
          dense={dense}
        />
      )}
      {showAiMoodView && (
        <MenuItemLink
          to={TAG_DASHBOARDS.aiMood.path}
          activeClassName={classes.active}
          primaryText={translate('resources.aiMood.name', { smart_count: 2 })}
          leftIcon={<MoodIcon />}
          sidebarIsOpen={open}
          dense={dense}
        />
      )}
      {showMyTagsView && (
        <MenuItemLink
          to={TAG_DASHBOARDS.myTags.path}
          activeClassName={classes.active}
          primaryText={translate('resources.myTags.name', { smart_count: 2 })}
          leftIcon={<LocalOfferIcon />}
          sidebarIsOpen={open}
          dense={dense}
        />
      )}
      {resources
        .filter(
          (r) =>
            r.name !== 'folder' &&
            r.name !== 'genre' &&
            (r.name !== 'podcastChannel' || showPodcasts) &&
            subItems(undefined)(r),
        )
        .map(renderResourceMenuItemLink)}
      {config.devSidebarPlaylists && open ? (
        <>
          <Divider />
          <PlaylistsSubMenu
            state={state}
            setState={setState}
            sidebarIsOpen={open}
            dense={dense}
          />
        </>
      ) : (
        resources.filter(subItems('playlist')).map(renderResourceMenuItemLink)
      )}
    </div>
  )
}

export default Menu
