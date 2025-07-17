import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import {
  useNotify,
  usePermissions,
  useTranslate,
  useDataProvider,
} from 'react-admin'
import { IconButton, Menu, MenuItem } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { MdQuestionMark } from 'react-icons/md'
import clsx from 'clsx'
import {
  playNext,
  addTracks,
  setTrack,
  openAddToPlaylist,
  openExtendedInfoDialog,
  openDownloadMenu,
  DOWNLOAD_MENU_SONG,
  openShareMenu,
} from '../actions'
import { LoveButton } from './LoveButton'
import config from '../config'
import { formatBytes } from '../utils'
import { useRedirect } from 'react-admin'

const useStyles = makeStyles({
  noWrap: {
    whiteSpace: 'nowrap',
  },
})

const MoreButton = ({ record, onClick, info }) => {
  const handleClick = record.missing
    ? (e) => {
        info.action(record)
        e.stopPropagation()
      }
    : onClick
  return (
    <IconButton onClick={handleClick} size={'small'}>
      {record?.missing ? (
        <MdQuestionMark fontSize={'large'} />
      ) : (
        <MoreVertIcon fontSize={'small'} />
      )}
    </IconButton>
  )
}

export const SongContextMenu = ({
  resource,
  record,
  showLove,
  onAddToPlaylist,
  className,
}) => {
  const classes = useStyles()
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const [anchorEl, setAnchorEl] = useState(null)
  const [playlistAnchorEl, setPlaylistAnchorEl] = useState(null)
  const [playlists, setPlaylists] = useState([])
  const [playlistsLoaded, setPlaylistsLoaded] = useState(false)
  const { permissions } = usePermissions()
  const redirect = useRedirect()

  const options = {
    playNow: {
      enabled: true,
      label: translate('resources.song.actions.playNow'),
      action: (record) => dispatch(setTrack(record)),
    },
    playNext: {
      enabled: true,
      label: translate('resources.song.actions.playNext'),
      action: (record) => dispatch(playNext({ [record.id]: record })),
    },
    addToQueue: {
      enabled: true,
      label: translate('resources.song.actions.addToQueue'),
      action: (record) => dispatch(addTracks({ [record.id]: record })),
    },
    addToPlaylist: {
      enabled: true,
      label: translate('resources.song.actions.addToPlaylist'),
      action: (record) =>
        dispatch(
          openAddToPlaylist({
            selectedIds: [record.mediaFileId || record.id],
            onSuccess: (id) => onAddToPlaylist(id),
          }),
        ),
    },
    showInPlaylist: {
      enabled: true,
      label:
        translate('resources.song.actions.showInPlaylist') +
        (playlists.length > 0 ? ' ►' : ''),
      action: (record, e) => {
        setPlaylistAnchorEl(e.currentTarget)
      },
    },
    share: {
      enabled: config.enableSharing,
      label: translate('ra.action.share'),
      action: (record) =>
        dispatch(
          openShareMenu(
            [record.mediaFileId || record.id],
            'song',
            record.title,
          ),
        ),
    },
    download: {
      enabled: config.enableDownloads,
      label: `${translate('ra.action.download')} (${formatBytes(record.size)})`,
      action: (record) =>
        dispatch(openDownloadMenu(record, DOWNLOAD_MENU_SONG)),
    },
    info: {
      enabled: true,
      label: translate('resources.song.actions.info'),
      action: async (record) => {
        let fullRecord = record
        if (permissions === 'admin' && !record.missing) {
          try {
            let id = record.mediaFileId ?? record.id
            const data = await dataProvider.inspect(id)
            fullRecord = { ...record, rawTags: data.data.rawTags }
          } catch (error) {
            notify(
              translate('ra.notification.http_error') + ': ' + error.message,
              {
                type: 'warning',
                multiLine: true,
                duration: 0,
              },
            )
          }
        }

        dispatch(openExtendedInfoDialog(fullRecord))
      },
    },
  }

  const handleClick = (e) => {
    setAnchorEl(e.currentTarget)
    if (!playlistsLoaded) {
      const id = record.mediaFileId || record.id
      dataProvider
        .getPlaylists(id)
        .then((res) => {
          setPlaylists(res.data)
          setPlaylistsLoaded(true)
        })
        .catch((error) => {
          // eslint-disable-next-line no-console
          console.error('Failed to fetch playlists:', error)
          setPlaylists([])
          setPlaylistsLoaded(true)
        })
    }
    e.stopPropagation()
  }

  const handleClose = (e) => {
    setAnchorEl(null)
    e.stopPropagation()
  }

  const handleItemClick = (e) => {
    e.preventDefault()
    const key = e.target.getAttribute('value')
    const action = options[key].action

    if (key === 'showInPlaylist') {
      // For showInPlaylist, we keep the main menu open and show submenu
      action(record, e)
    } else {
      // For other actions, close the main menu
      setAnchorEl(null)
      action(record)
    }
    e.stopPropagation()
  }

  const handlePlaylistClose = (e) => {
    setPlaylistAnchorEl(null)
    if (e) {
      e.stopPropagation()
    }
  }

  const handleMainMenuClose = (e) => {
    setAnchorEl(null)
    setPlaylistAnchorEl(null) // Close both menus
    e.stopPropagation()
  }

  const handlePlaylistClick = (id, e) => {
    e.stopPropagation()
    redirect(`/playlist/${id}/show`)
    handlePlaylistClose()
  }

  const open = Boolean(anchorEl)

  if (!record) {
    return null
  }

  const present = !record.missing

  return (
    <span className={clsx(classes.noWrap, className)}>
      <LoveButton
        record={record}
        resource={resource}
        visible={config.enableFavourites && showLove && present}
      />
      <MoreButton record={record} onClick={handleClick} info={options.info} />
      <Menu
        id={'menu' + record.id}
        anchorEl={anchorEl}
        open={open}
        onClose={handleMainMenuClose}
      >
        {Object.keys(options).map((key) => {
          const showInPlaylistDisabled =
            key === 'showInPlaylist' && !playlists.length
          return (
            options[key].enabled && (
              <MenuItem
                value={key}
                key={key}
                onClick={
                  showInPlaylistDisabled
                    ? (e) => e.stopPropagation()
                    : handleItemClick
                }
                disabled={showInPlaylistDisabled}
                style={
                  showInPlaylistDisabled ? { pointerEvents: 'auto' } : undefined
                }
              >
                {options[key].label}
              </MenuItem>
            )
          )
        })}
      </Menu>
      <Menu
        anchorEl={playlistAnchorEl}
        open={Boolean(playlistAnchorEl)}
        onClose={handlePlaylistClose}
        anchorOrigin={{
          vertical: 'top',
          horizontal: 'right',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'left',
        }}
      >
        {playlists.map((p) => (
          <MenuItem key={p.id} onClick={(e) => handlePlaylistClick(p.id, e)}>
            {p.name}
          </MenuItem>
        ))}
      </Menu>
    </span>
  )
}

SongContextMenu.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object.isRequired,
  onAddToPlaylist: PropTypes.func,
  showLove: PropTypes.bool,
}

SongContextMenu.defaultProps = {
  onAddToPlaylist: () => {},
  record: {},
  resource: 'song',
  showLove: true,
  addLabel: true,
}
