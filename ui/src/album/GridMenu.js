import React from 'react'
import IconButton from '@material-ui/core/IconButton'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { useDispatch } from 'react-redux'
import { playAlbum, shuffleAlbum } from '../audioplayer'
import { useGetList } from 'react-admin'

const GridMenu = (props) => {
  const [anchorEl, setAnchorEl] = React.useState(null)
  const dispatch = useDispatch()
  const { ids, data, loading, error } = useGetList(
    'albumSong',
    {},
    { field: 'trackNumber', order: 'ASC' },
    { album_id: props.id }
  )

  if (loading) {
    return (
      <IconButton>
        <MoreVertIcon />
      </IconButton>
    )
  }

  if (error) {
    return (
      <IconButton>
        <MoreVertIcon />
      </IconButton>
    )
  }

  const options = [
    [1, 'Play'],
    [2, 'Shuffle'],
  ]

  const open = Boolean(anchorEl)

  const handleClick = (e) => {
    e.preventDefault()
    setAnchorEl(e.currentTarget)
  }

  const handleClose = (e) => {
    e.preventDefault()
    setAnchorEl(null)
  }

  const handleItemClick = (e) => {
    e.preventDefault()
    setAnchorEl(null)
    const value = e.target.getAttribute('value')
    if (value === '1') {
      dispatch(playAlbum(ids[0], data))
    }
    if (value === '2') {
      dispatch(shuffleAlbum(data))
    }
  }

  return (
    <div>
      <IconButton
        aria-label="more"
        aria-controls="long-menu"
        aria-haspopup="true"
        onClick={handleClick}
      >
        <MoreVertIcon />
      </IconButton>
      <Menu
        id="long-menu"
        anchorEl={anchorEl}
        keepMounted
        open={open}
        onClose={handleClose}
      >
        {options.map((option) => (
          <MenuItem value={option[0]} key={option[0]} onClick={handleItemClick}>
            {option[1]}
          </MenuItem>
        ))}
      </Menu>
    </div>
  )
}
export default GridMenu
