import React, { useState } from 'react'
import IconButton from '@material-ui/core/IconButton'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import Checkbox from '@material-ui/core/Checkbox'
import { useDispatch, useSelector } from 'react-redux'
import { useTranslate } from 'react-admin'
import { setToggleableFields } from '../actions'

const ITEM_HEIGHT = 70

export default function ToggleFieldsMenu({ resource }) {
  const [anchorEl, setAnchorEl] = useState(null)
  const dispatch = useDispatch()
  const translate = useTranslate()
  const toggleableColumns = useSelector(
    (state) => state.settings.toggleableFields[resource]
  )

  const open = Boolean(anchorEl)

  const handleOpen = (event) => {
    setAnchorEl(event.currentTarget)
  }
  const handleClose = () => {
    setAnchorEl(null)
  }

  const handleClick = (selectedColumn) => {
    dispatch(
      setToggleableFields({
        [resource]: {
          ...toggleableColumns,
          [selectedColumn]: !toggleableColumns[selectedColumn],
        },
      })
    )
  }

  if (!toggleableColumns || !Object.keys(toggleableColumns).length) {
    return null
  }

  return (
    <div style={{ position: 'relative', top: '-0.5em' }}>
      <IconButton
        aria-label="more"
        aria-controls="long-menu"
        aria-haspopup="true"
        onClick={handleOpen}
      >
        <MoreVertIcon />
      </IconButton>
      <Menu
        id="long-menu"
        anchorEl={anchorEl}
        keepMounted
        open={open}
        onClose={handleClose}
        PaperProps={{
          style: {
            maxHeight: ITEM_HEIGHT * 4.5,
            width: '20ch',
          },
        }}
      >
        {Object.entries(toggleableColumns).map(([key, val]) => (
          <MenuItem key={key} onClick={() => handleClick(key)}>
            <Checkbox checked={val} />
            {translate(`resources.${resource}.fields.${key}`)}
          </MenuItem>
        ))}
      </Menu>
    </div>
  )
}
