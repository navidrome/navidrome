import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import IconButton from '@material-ui/core/IconButton'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { makeStyles } from '@material-ui/core/styles'
import { useDataProvider, useNotify, useTranslate } from 'react-admin'
import clsx from 'clsx'
import {
  playNext,
  addTracks,
  playTracks,
  shuffleTracks,
  openAddToPlaylist,
  openDownloadMenu,
  openExtendedInfoDialog,
  DOWNLOAD_MENU_ALBUM,
  DOWNLOAD_MENU_ARTIST,
  openShareMenu,
} from '../actions'
import { LoveButton } from './LoveButton'
import config from '../config'
import { formatBytes } from '../utils'

const useStyles = makeStyles({
  noWrap: {
    whiteSpace: 'nowrap',
  },
  menu: {
    color: (props) => props.color,
  },
})

const ContextMenu = ({
  resource,
  showLove,
  record,
  color,
  className,
  songQueryParams,
  hideInfo,
}) => {
  const classes = useStyles({ color })
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const [anchorEl, setAnchorEl] = useState(null)

  const options = {
    play: {
      enabled: true,
      needData: true,
      label: translate('resources.album.actions.playAll'),
      action: (data, ids) => dispatch(playTracks(data, ids)),
    },
    playNext: {
      enabled: true,
      needData: true,
      label: translate('resources.album.actions.playNext'),
      action: (data, ids) => dispatch(playNext(data, ids)),
    },
    addToQueue: {
      enabled: true,
      needData: true,
      label: translate('resources.album.actions.addToQueue'),
      action: (data, ids) => dispatch(addTracks(data, ids)),
    },
    shuffle: {
      enabled: true,
      needData: true,
      label: translate('resources.album.actions.shuffle'),
      action: (data, ids) => dispatch(shuffleTracks(data, ids)),
    },
    addToPlaylist: {
      enabled: true,
      needData: true,
      label: translate('resources.album.actions.addToPlaylist'),
      action: (data, ids) => dispatch(openAddToPlaylist({ selectedIds: ids })),
    },
    share: {
      enabled: config.enableSharing,
      needData: false,
      label: translate('ra.action.share'),
      action: (record) =>
        dispatch(openShareMenu([record.id], resource, record.name)),
    },
    download: {
      enabled: config.enableDownloads && record.size,
      needData: false,
      label: `${translate('ra.action.download')} (${formatBytes(record.size)})`,
      action: () => {
        dispatch(
          openDownloadMenu(
            record,
            record.duration !== undefined
              ? DOWNLOAD_MENU_ALBUM
              : DOWNLOAD_MENU_ARTIST,
          ),
        )
      },
    },
    ...(!hideInfo && {
      info: {
        enabled: true,
        needData: true,
        label: translate('resources.album.actions.info'),
        action: () => dispatch(openExtendedInfoDialog(record)),
      },
    }),
  }

  const handleClick = (e) => {
    e.preventDefault()
    setAnchorEl(e.currentTarget)
    e.stopPropagation()
  }

  const handleOnClose = (e) => {
    e.preventDefault()
    setAnchorEl(null)
    e.stopPropagation()
  }

  let extractSongsData = function (response) {
    const data = response.data.reduce(
      (acc, cur) => ({ ...acc, [cur.id]: cur }),
      {},
    )
    const ids = response.data.map((r) => r.id)
    return { data, ids }
  }

  const handleItemClick = (e) => {
    setAnchorEl(null)
    const key = e.target.getAttribute('value')
    if (options[key].needData) {
      dataProvider
        .getList('song', songQueryParams)
        .then((response) => {
          let { data, ids } = extractSongsData(response)
          options[key].action(data, ids)
        })
        .catch(() => {
          notify('ra.page.error', 'warning')
        })
    } else {
      options[key].action(record)
    }

    e.stopPropagation()
  }

  const open = Boolean(anchorEl)

  return (
    <span className={clsx(classes.noWrap, className)}>
      <LoveButton
        record={record}
        resource={resource}
        visible={config.enableFavourites && showLove}
        color={color}
      />
      <IconButton
        aria-label="more"
        aria-controls="context-menu"
        aria-haspopup="true"
        className={classes.menu}
        onClick={handleClick}
        size={'small'}
      >
        <MoreVertIcon fontSize={'small'} />
      </IconButton>
      <Menu
        id="context-menu"
        anchorEl={anchorEl}
        keepMounted
        open={open}
        onClose={handleOnClose}
      >
        {Object.keys(options).map(
          (key) =>
            options[key].enabled && (
              <MenuItem value={key} key={key} onClick={handleItemClick}>
                {options[key].label}
              </MenuItem>
            ),
        )}
      </Menu>
    </span>
  )
}

export const AlbumContextMenu = (props) =>
  props.record ? (
    <ContextMenu
      {...props}
      resource={'album'}
      songQueryParams={{
        pagination: { page: 1, perPage: -1 },
        sort: { field: 'releaseDate, discNumber, trackNumber', order: 'ASC' },
        filter: {
          album_id: props.record.id,
          release_date: props.releaseDate,
          disc_number: props.discNumber,
        },
      }}
    />
  ) : null

AlbumContextMenu.propTypes = {
  record: PropTypes.object,
  discNumber: PropTypes.number,
  color: PropTypes.string,
  showLove: PropTypes.bool,
}

AlbumContextMenu.defaultProps = {
  showLove: true,
  addLabel: true,
}

export const ArtistContextMenu = (props) =>
  props.record ? (
    <ContextMenu
      {...props}
      hideInfo={true}
      resource={'artist'}
      songQueryParams={{
        pagination: { page: 1, perPage: 200 },
        sort: {
          field: 'album, releaseDate, discNumber, trackNumber',
          order: 'ASC',
        },
        filter: { album_artist_id: props.record.id },
      }}
    />
  ) : null

ArtistContextMenu.propTypes = {
  record: PropTypes.object,
  color: PropTypes.string,
  showLove: PropTypes.bool,
}

ArtistContextMenu.defaultProps = {
  showLove: true,
  addLabel: true,
}
