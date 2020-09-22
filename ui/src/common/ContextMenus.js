import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import IconButton from '@material-ui/core/IconButton'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { makeStyles } from '@material-ui/core/styles'
import { useDataProvider, useNotify, useTranslate } from 'react-admin'
import { playNext, addTracks, playTracks, shuffleTracks } from '../audioplayer'
import { openAddToPlaylist } from '../dialogs/dialogState'
import subsonic from '../subsonic'
import StarButton from './StarButton'

const useStyles = makeStyles({
  noWrap: {
    whiteSpace: 'nowrap',
  },
  menu: {
    color: (props) => props.color,
    visibility: (props) => (props.visible ? 'visible' : 'hidden'),
  },
})

const ContextMenu = ({
  resource,
  showStar,
  record,
  color,
  visible,
  songQueryParams,
}) => {
  const classes = useStyles({ color, visible })
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const translate = useTranslate()
  const notify = useNotify()
  const [anchorEl, setAnchorEl] = useState(null)

  const options = {
    play: {
      needData: true,
      label: 'resources.album.actions.playAll',
      action: (data, ids) => dispatch(playTracks(data, ids)),
    },
    playNext: {
      needData: true,
      label: 'resources.album.actions.playNext',
      action: (data, ids) => dispatch(playNext(data, ids)),
    },
    addToQueue: {
      needData: true,
      label: 'resources.album.actions.addToQueue',
      action: (data, ids) => dispatch(addTracks(data, ids)),
    },
    shuffle: {
      needData: true,
      label: 'resources.album.actions.shuffle',
      action: (data, ids) => dispatch(shuffleTracks(data, ids)),
    },
    addToPlaylist: {
      needData: true,
      label: 'resources.album.actions.addToPlaylist',
      action: (data, ids) => dispatch(openAddToPlaylist({ selectedIds: ids })),
    },
    download: {
      needData: false,
      label: 'resources.album.actions.download',
      action: () => subsonic.download(record.id),
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
    if (options[key].needData) {
      dataProvider
        .getList('albumSong', songQueryParams)
        .then((response) => {
          let { data, ids } = extractSongsData(response)
          options[key].action(data, ids)
        })
        .catch(() => {
          notify('ra.page.error', 'warning')
        })
    } else {
      options[key].action()
    }

    e.stopPropagation()
  }

  const open = Boolean(anchorEl)

  return (
    <span className={classes.noWrap}>
      <StarButton
        record={record}
        resource={resource}
        visible={visible && showStar}
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
        {Object.keys(options).map((key) => (
          <MenuItem value={key} key={key} onClick={handleItemClick}>
            {translate(options[key].label)}
          </MenuItem>
        ))}
      </Menu>
    </span>
  )
}

export const AlbumContextMenu = (props) => (
  <ContextMenu
    {...props}
    resource={'album'}
    songQueryParams={{
      pagination: { page: 1, perPage: -1 },
      sort: { field: 'discNumber, trackNumber', order: 'ASC' },
      filter: { album_id: props.record.id, disc_number: props.discNumber },
    }}
  />
)

AlbumContextMenu.propTypes = {
  record: PropTypes.object,
  discNumber: PropTypes.number,
  visible: PropTypes.bool,
  color: PropTypes.string,
  showStar: PropTypes.bool,
}

AlbumContextMenu.defaultProps = {
  visible: true,
  showStar: true,
  addLabel: true,
}

export const ArtistContextMenu = (props) => (
  <ContextMenu
    {...props}
    resource={'artist'}
    songQueryParams={{
      pagination: { page: 1, perPage: 200 },
      sort: { field: 'album, discNumber, trackNumber', order: 'ASC' },
      filter: { album_artist_id: props.record.id },
    }}
  />
)

ArtistContextMenu.propTypes = {
  record: PropTypes.object,
  visible: PropTypes.bool,
  color: PropTypes.string,
  showStar: PropTypes.bool,
}

ArtistContextMenu.defaultProps = {
  visible: true,
  showStar: true,
  addLabel: true,
}
