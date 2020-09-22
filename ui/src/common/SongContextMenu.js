import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import { useTranslate } from 'react-admin'
import { IconButton, Menu, MenuItem } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { playNext, addTracks, setTrack } from '../audioplayer'
import { openAddToPlaylist } from '../dialogs/dialogState'
import subsonic from '../subsonic'
import StarButton from './StarButton'

const useStyles = makeStyles({
  noWrap: {
    whiteSpace: 'nowrap',
  },
  menu: {
    visibility: (props) => (props.visible ? 'visible' : 'hidden'),
  },
})

const SongContextMenu = ({
  resource,
  record,
  showStar,
  onAddToPlaylist,
  visible,
}) => {
  const classes = useStyles({ visible })
  const dispatch = useDispatch()
  const translate = useTranslate()
  const [anchorEl, setAnchorEl] = useState(null)
  const options = {
    playNow: {
      label: 'resources.song.actions.playNow',
      action: (record) => dispatch(setTrack(record)),
    },
    playNext: {
      label: 'resources.song.actions.playNext',
      action: (record) => dispatch(playNext({ [record.id]: record })),
    },
    addToQueue: {
      label: 'resources.song.actions.addToQueue',
      action: (record) => dispatch(addTracks({ [record.id]: record })),
    },
    addToPlaylist: {
      label: 'resources.song.actions.addToPlaylist',
      action: (record) =>
        dispatch(
          openAddToPlaylist({
            selectedIds: [record.mediaFileId || record.id],
            onSuccess: (id) => onAddToPlaylist(id),
          })
        ),
    },
    download: {
      label: 'resources.song.actions.download',
      action: (record) => subsonic.download(record.id),
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

  return (
    <span className={classes.noWrap}>
      {showStar && (
        <StarButton record={record} resource={resource} visible={visible} />
      )}
      <IconButton onClick={handleClick} size={'small'} className={classes.menu}>
        <MoreVertIcon fontSize={'small'} />
      </IconButton>
      <Menu
        id={'menu' + record.id}
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
      >
        {Object.keys(options).map((key) => (
          <MenuItem value={key} key={key} onClick={handleItemClick}>
            {translate(options[key].label)}
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
  visible: PropTypes.bool,
  showStar: PropTypes.bool,
}

SongContextMenu.defaultProps = {
  onAddToPlaylist: () => {},
  record: {},
  resource: 'song',
  visible: true,
  showStar: true,
  addLabel: true,
}

export default SongContextMenu
