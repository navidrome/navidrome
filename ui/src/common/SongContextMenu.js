import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import { useUpdate, useTranslate, useRefresh, useNotify } from 'react-admin'
import { IconButton, Menu, MenuItem } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import StarIcon from '@material-ui/icons/Star'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import { addTracks, setTrack } from '../audioplayer'
import config from '../config'
import { openAddToPlaylist } from '../dialogs/dialogState'

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

const SongContextMenu = ({
  resource,
  record,
  showStar,
  onAddToPlaylist,
  visible,
}) => {
  const classes = useStyles({ visible, starred: record.starred })
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const [anchorEl, setAnchorEl] = useState(null)
  const options = {
    playNow: {
      label: 'resources.song.actions.playNow',
      action: (record) => setTrack(record),
    },
    addToQueue: {
      label: 'resources.song.actions.addToQueue',
      action: (record) => addTracks({ [record.id]: record }),
    },
    addToPlaylist: {
      label: 'resources.song.actions.addToPlaylist',
      action: (record) =>
        openAddToPlaylist({
          selectedIds: [record.mediaFileId || record.id],
          onSuccess: (id) => onAddToPlaylist(id),
        }),
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

  const [toggleStarred, { loading: updating }] = useUpdate(
    resource,
    record.id,
    {
      ...record,
      starred: !record.starred,
    },
    {
      undoable: false,
      onFailure: (error) => {
        console.log(error)
        notify('ra.page.error', 'warning')
        refresh()
      },
    }
  )

  const handleToggleStar = (e) => {
    toggleStarred()
    e.stopPropagation()
  }

  const open = Boolean(anchorEl)

  return (
    <span className={classes.noWrap}>
      {config.enableStarred && showStar && (
        <IconButton
          onClick={handleToggleStar}
          size={'small'}
          disabled={updating}
          className={classes.star}
        >
          {record.starred ? (
            <StarIcon fontSize={'small'} />
          ) : (
            <StarBorderIcon fontSize={'small'} />
          )}
        </IconButton>
      )}
      <IconButton
        onClick={handleClick}
        size={'small'}
        className={classes.menu}
        disabled={updating}
      >
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
  resource: PropTypes.string,
  record: PropTypes.object,
  onAddToPlaylist: PropTypes.func,
  visible: PropTypes.bool,
  showStar: PropTypes.bool,
}

SongContextMenu.defaultProps = {
  onAddToPlaylist: () => {},
  visible: true,
  showStar: true,
  addLabel: true,
}

export default SongContextMenu
