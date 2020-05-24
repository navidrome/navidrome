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
import AddToPlaylistMenu from './AddToPlaylistMenu'
import config from '../config'

const useStyles = makeStyles({
  noWrap: {
    whiteSpace: 'nowrap',
  },
  menu: {
    visibility: (props) => (props.visible ? 'visible' : 'hidden'),
  },
  star: {
    visibility: (props) =>
      props.visible || props.starred ? 'visible' : 'hidden',
  },
})

const SongContextMenu = ({ record, showStar, onAddToPlaylist, visible }) => {
  const classes = useStyles({ visible, starred: record.starred })
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

  const [updateRecord, { loading: updating }] = useUpdate(
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

  const toggleStar = (record) => {
    record.starred = !record.starred
    updateRecord()
  }

  const handleToggleStar = (e, record) => {
    toggleStar(record)
    e.stopPropagation()
  }

  const open = Boolean(anchorEl)

  return (
    <span className={classes.noWrap}>
      {config.enableStarred && showStar && (
        <IconButton
          onClick={(e) => handleToggleStar(e, record)}
          size={'small'}
          disabled={updating}
          className={classes.star}
        >
          {record.starred ? <StarIcon /> : <StarBorderIcon />}
        </IconButton>
      )}
      <IconButton
        onClick={handleClick}
        size={'small'}
        className={classes.menu}
        disabled={updating}
      >
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
  visible: PropTypes.bool,
  showStar: PropTypes.bool,
}

SongContextMenu.defaultProps = {
  visible: true,
  showStar: true,
  addLabel: true,
}

export default SongContextMenu
