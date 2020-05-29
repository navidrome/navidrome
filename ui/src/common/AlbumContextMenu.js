import React, { useState } from 'react'
import { useDispatch } from 'react-redux'
import IconButton from '@material-ui/core/IconButton'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { makeStyles } from '@material-ui/core/styles'
import { useDataProvider, useNotify, useTranslate } from 'react-admin'
import { addTracks, playTracks, shuffleTracks } from '../audioplayer'
import { openAddToPlaylist } from '../dialogs/dialogState'
import StarIcon from '@material-ui/icons/Star'
import PropTypes from 'prop-types'

const useStyles = makeStyles({
  noWrap: {
    whiteSpace: 'nowrap',
  },
  menu: {
    color: (props) => props.color,
    visibility: (props) => (props.visible ? 'visible' : 'hidden'),
  },
  star: {
    visibility: 'hidden', // TODO: Invisible for now
  },
})

const AlbumContextMenu = ({ record, discNumber, color, visible }) => {
  const classes = useStyles({ color, visible })
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const [anchorEl, setAnchorEl] = useState(null)

  const options = {
    play: {
      label: 'resources.album.actions.playAll',
      action: playTracks,
    },
    addToQueue: {
      label: 'resources.album.actions.addToQueue',
      action: addTracks,
    },
    shuffle: {
      label: 'resources.album.actions.shuffle',
      action: shuffleTracks,
    },
    addToPlaylist: {
      label: 'resources.song.actions.addToPlaylist',
      action: (data, ids) => openAddToPlaylist({ selectedIds: ids }),
    },
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
      {}
    )
    const ids = response.data.map((r) => r.id)
    return { data, ids }
  }

  const handleItemClick = (e) => {
    setAnchorEl(null)
    const key = e.target.getAttribute('value')
    dataProvider
      .getList('albumSong', {
        pagination: { page: 1, perPage: -1 },
        sort: { field: 'discNumber, trackNumber', order: 'ASC' },
        filter: { album_id: record.id, disc_number: discNumber },
      })
      .then((response) => {
        let { data, ids } = extractSongsData(response)
        dispatch(options[key].action(data, ids))
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })

    e.stopPropagation()
  }

  const open = Boolean(anchorEl)

  return (
    <span className={classes.noWrap}>
      <IconButton size={'small'} className={classes.star}>
        <StarIcon fontSize={'small'} />
      </IconButton>
      <IconButton
        aria-label="more"
        aria-controls="context-menu"
        aria-haspopup="true"
        className={classes.menu}
        onClick={handleClick}
        size={'small'}
      >
        <MoreVertIcon />
      </IconButton>
      <Menu
        id="context-menu"
        anchorEl={anchorEl}
        keepMounted
        open={open}
        onClose={handleOnClose}
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

AlbumContextMenu.propTypes = {
  record: PropTypes.object,
  discNumber: PropTypes.number,
  visible: PropTypes.bool,
  color: PropTypes.string,
}

AlbumContextMenu.defaultProps = {
  visible: true,
  addLabel: true,
}

export default AlbumContextMenu
