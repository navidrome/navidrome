import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import { useNotify, usePermissions, useTranslate } from 'react-admin'
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
import { httpClient } from '../dataProvider'

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
  const [anchorEl, setAnchorEl] = useState(null)
  const { permissions } = usePermissions()

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
            const data = await httpClient(`/api/inspect?id=${id}`)
            fullRecord = { ...record, rawTags: data.json.rawTags }
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
    e.stopPropagation()
  }

  const handleClose = (e) => {
    setAnchorEl(null)
    e.stopPropagation()
  }

  const handleItemClick = (e) => {
    e.preventDefault()
    setAnchorEl(null)
    const key = e.target.getAttribute('value')
    options[key].action(record)
    e.stopPropagation()
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
        onClose={handleClose}
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
