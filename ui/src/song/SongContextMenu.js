import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import { useUpdate, useTranslate, useRefresh, useNotify } from 'react-admin'
import { IconButton, Menu, MenuItem } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import StarIcon from '@material-ui/icons/Star'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import NestedMenuItem from 'material-ui-nested-menu-item'
import { addTracks, setTrack } from '../audioplayer'
import { AddToPlaylistMenu } from '../common'
import config from '../config'

const useStyles = makeStyles({
  noWrap: {
    whiteSpace: 'nowrap',
  },
})

export const SongContextMenu = ({ className, record, onAddToPlaylist }) => {
  const classes = useStyles()
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const [anchorEl, setAnchorEl] = useState(null)
  const options = {
    playNow: {
      label: translate('resources.song.actions.playNow'),
      action: (record) => setTrack(record),
    },
    addToQueue: {
      label: translate('resources.song.actions.addToQueue'),
      action: (record) => addTracks({ [record.id]: record }),
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
    dispatch(options[key].action(record))
    e.stopPropagation()
  }

  const [toggleStar, { toggling: loading }] = useUpdate(
    'albumSong',
    record.id,
    record,
    {
      undoable: false,
      onFailure: (error) => {
        console.log(error)
        notify('ra.page.error', 'warning')
        refresh()
      },
    }
  )

  const handleToggleStar = (e, record) => {
    record.starred = !record.starred
    toggleStar()
    e.stopPropagation()
  }

  const open = Boolean(anchorEl)

  return (
    <span className={`${classes.noWrap} ${className}`}>
      {config.enableStarred && (
        <IconButton
          onClick={(e) => handleToggleStar(e, record)}
          size={'small'}
          disabled={loading}
        >
          {record.starred ? <StarIcon /> : <StarBorderIcon />}
        </IconButton>
      )}
      <IconButton onClick={handleClick} size={'small'}>
        <MoreVertIcon />
      </IconButton>
      <Menu
        id={'menu' + record.id}
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
      >
        {Object.keys(options).map((key) => (
          <MenuItem value={key} key={key} onClick={handleItemClick}>
            {options[key].label}
          </MenuItem>
        ))}
        <NestedMenuItem
          label={translate('resources.song.actions.addToPlaylist')}
          parentMenuOpen={open}
        >
          <AddToPlaylistMenu
            selectedIds={[record.mediaFileId || record.id]}
            onClose={handleClose}
            onItemAdded={onAddToPlaylist}
          />
        </NestedMenuItem>
      </Menu>
    </span>
  )
}

SongContextMenu.propTypes = {
  record: PropTypes.object,
  onAddToPlaylist: PropTypes.func,
}
