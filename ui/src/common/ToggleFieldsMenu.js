import React, { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import IconButton from '@material-ui/core/IconButton'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import { makeStyles, Typography } from '@material-ui/core'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import Checkbox from '@material-ui/core/Checkbox'
import { useDispatch, useSelector } from 'react-redux'
import { useTranslate } from 'react-admin'
import { setToggleableFields, albumViewList } from '../actions'

const useStyles = makeStyles({
  menuIcon: {
    position: 'relative',
    top: '-0.5em',
  },
  menu: {
    width: '24ch',
  },
  columns: {
    maxHeight: '21rem',
    overflow: 'auto',
  },
  title: {
    margin: '1rem',
  },
})

const ToggleFieldsMenu = ({ resource, TopBarComponent }) => {
  const [anchorEl, setAnchorEl] = useState(null)
  const dispatch = useDispatch()
  const translate = useTranslate()
  const toggleableColumns = useSelector(
    (state) => state.settings.toggleableFields[resource]
  )
  const omittedColumns =
    useSelector((state) => state.settings.omittedFields[resource]) || []

  const classes = useStyles()
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

  useEffect(() => {
    if (!toggleableColumns && resource === 'album') {
      dispatch(albumViewList())
    }
  }, [dispatch, resource, toggleableColumns])

  return (
    <div className={classes.menuIcon}>
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
        classes={{
          paper: classes.menu,
        }}
      >
        {TopBarComponent && <TopBarComponent />}
        <Typography className={classes.title}>
          {translate('ra.toggleFieldsMenu.columnsToDisplay')}
        </Typography>
        <div className={classes.columns}>
          {toggleableColumns &&
            Object.entries(toggleableColumns).map(([key, val]) =>
              !omittedColumns.includes(key) ? (
                <MenuItem key={key} onClick={() => handleClick(key)}>
                  <Checkbox checked={val} />
                  {translate(`resources.${resource}.fields.${key}`)}
                </MenuItem>
              ) : null
            )}
        </div>
      </Menu>
    </div>
  )
}

export default ToggleFieldsMenu

ToggleFieldsMenu.propTypes = {
  resource: PropTypes.string.isRequired,
  TopBarComponent: PropTypes.element,
}
