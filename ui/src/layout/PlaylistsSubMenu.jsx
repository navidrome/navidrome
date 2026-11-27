import React, { useCallback, useMemo, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import {
  MenuItemLink,
  useDataProvider,
  useNotify,
  useQueryWithStore,
  useTranslate,
} from 'react-admin'
import { useHistory } from 'react-router-dom'
import QueueMusicIcon from '@material-ui/icons/QueueMusic'
import { Typography } from '@material-ui/core'
import QueueMusicOutlinedIcon from '@material-ui/icons/QueueMusicOutlined'
import FavoriteIcon from '@material-ui/icons/Favorite'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import { BiListUl } from 'react-icons/bi'
import { useDrop } from 'react-dnd'
import SubMenu from './SubMenu'
import { canChangeTracks, OverflowTooltip, useRefreshOnEvents } from '../common'
import { DraggableTypes } from '../consts'
import { setSidebarPlaylistsOnlyFavourites } from '../actions'
import config from '../config'

const PlaylistMenuItemLink = ({ pls, sidebarIsOpen }) => {
  const dataProvider = useDataProvider()
  const notify = useNotify()

  const [, dropRef] = useDrop(() => ({
    accept: canChangeTracks(pls) ? DraggableTypes.ALL : [],
    drop: (item) =>
      dataProvider
        .addToPlaylist(pls.id, item)
        .then((res) => {
          notify('message.songsAddedToPlaylist', 'info', {
            smart_count: res.data?.added,
          })
        })
        .catch(() => {
          notify('ra.page.error', 'warning')
        }),
  }))

  return (
    <MenuItemLink
      to={`/playlist/${pls.id}/show`}
      primaryText={
        <OverflowTooltip title={pls.name} placement="right">
          <Typography variant="inherit" noWrap ref={dropRef}>
            {pls.name}
          </Typography>
        </OverflowTooltip>
      }
      sidebarIsOpen={sidebarIsOpen}
      dense={false}
    />
  )
}

const PlaylistsSubMenu = ({ state, setState, sidebarIsOpen, dense }) => {
  const history = useHistory()
  const dispatch = useDispatch()
  const translate = useTranslate()
  const onlyFavourites = useSelector(
    (state) => state.settings.sidebarPlaylistsOnlyFavourites,
  )
  // Ignore a persisted preference when the feature is off, so disabling it later
  // (with the toggle now hidden) doesn't strand the user on a filtered sidebar
  const showFavouritesOnly = config.enableFavourites && onlyFavourites
  const playlistData = useSelector(
    (state) => state.admin.resources.playlist?.data,
  )
  // Fingerprint of local star state; changes only when a playlist is (un)starred,
  // so a local toggle refetches the sidebar without the SSE echo the actor never gets
  const starFingerprint = useMemo(() => {
    const data = playlistData || {}
    return Object.keys(data)
      .filter((id) => data[id]?.starred)
      .sort()
      .join(',')
  }, [playlistData])
  const [refreshCount, setRefreshCount] = useState(0)

  // Only the favourites-only view depends on star state changing elsewhere;
  // when showing all playlists a star event from another client changes nothing
  // async because useRefreshOnEvents calls .catch() on the returned value
  const onRefresh = useCallback(async () => {
    if (showFavouritesOnly) setRefreshCount((count) => count + 1)
  }, [showFavouritesOnly])
  useRefreshOnEvents({ events: ['playlist'], onRefresh })

  // A changed payload signature makes useQueryWithStore refetch
  const { data, loaded } = useQueryWithStore({
    type: 'getList',
    resource: 'playlist',
    payload: {
      pagination: {
        page: 0,
        perPage: config.maxSidebarPlaylists,
      },
      sort: { field: 'name' },
      ...(showFavouritesOnly && {
        filter: { starred: true },
        starFingerprint,
        refresh: refreshCount,
      }),
    },
  })

  const handleToggle = (menu) => {
    setState((state) => ({ ...state, [menu]: !state[menu] }))
  }

  const renderPlaylistMenuItemLink = (pls) => (
    <PlaylistMenuItemLink
      pls={pls}
      sidebarIsOpen={sidebarIsOpen}
      key={pls.id}
    />
  )

  const userId = localStorage.getItem('userId')
  const myPlaylists = []
  const sharedPlaylists = []

  if (loaded && data) {
    const allPlaylists = Object.keys(data).map((id) => data[id])

    allPlaylists.forEach((pls) => {
      if (userId === pls.ownerId) {
        myPlaylists.push(pls)
      } else {
        sharedPlaylists.push(pls)
      }
    })
  }

  const onPlaylistConfig = useCallback(
    () => history.push('/playlist'),
    [history],
  )

  const handleToggleFavourites = useCallback(() => {
    dispatch(setSidebarPlaylistsOnlyFavourites(!onlyFavourites))
  }, [dispatch, onlyFavourites])

  return (
    <>
      <SubMenu
        handleToggle={() => handleToggle('menuPlaylists')}
        isOpen={state.menuPlaylists}
        sidebarIsOpen={sidebarIsOpen}
        name={'menu.playlists'}
        icon={<QueueMusicIcon />}
        dense={dense}
        actionIcon={<BiListUl />}
        onAction={onPlaylistConfig}
        onSecondaryAction={
          config.enableFavourites ? handleToggleFavourites : undefined
        }
        secondaryActionIcon={
          onlyFavourites ? (
            <FavoriteIcon fontSize={'small'} />
          ) : (
            <FavoriteBorderIcon fontSize={'small'} />
          )
        }
        secondaryActionTitle={translate('menu.onlyFavourites')}
        secondaryActionActive={onlyFavourites}
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
