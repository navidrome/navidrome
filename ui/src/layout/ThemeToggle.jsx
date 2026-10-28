import React, { useState, useEffect } from 'react'
import { IconButton, Tooltip, Menu, MenuItem, ListItemIcon, ListItemText, Divider } from '@mui/material'
import { useDispatch, useSelector } from 'react-redux'
import Brightness4Icon from '@mui/icons-material/Brightness4'
import Brightness7Icon from '@mui/icons-material/Brightness7'
import { changeTheme } from '../actions'
import { AUTO_THEME_ID, THEME_IDS } from '../consts'
import themes from '../themes'
import { useTranslate } from 'react-admin'

const DARK_THEME = 'DarkTheme'
const LIGHT_THEME = 'LightTheme'

export default function ThemeToggle() {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const currentTheme = useSelector((state) => state.theme)
  const [anchorEl, setAnchorEl] = useState(null)

  const isDark = currentTheme === DARK_THEME
  const isLight = currentTheme === LIGHT_THEME
  const isAuto = currentTheme === AUTO_THEME_ID

  const resolvedDark = isAuto
    ? (themes.DarkTheme?.palette?.type === 'dark')
    : (themes[currentTheme]?.palette?.type === 'dark')

  const handleToggle = () => {
    if (isAuto) {
      dispatch(changeTheme(resolvedDark ? LIGHT_THEME : DARK_THEME))
    } else if (isDark) {
      dispatch(changeTheme(LIGHT_THEME))
    } else {
      dispatch(changeTheme(DARK_THEME))
    }
    setAnchorEl(null)
  }

  const handleAuto = () => {
    dispatch(changeTheme(AUTO_THEME_ID))
    setAnchorEl(null)
  }

  return (
    <>
      <Tooltip title={translate('theme.toggle')}>
        <IconButton
          color="inherit"
          onClick={(e) => setAnchorEl(e.currentTarget)}
          aria-label="Toggle theme"
        >
          {resolvedDark ? <Brightness4Icon /> : <Brightness7Icon />}
        </IconButton>
      </Tooltip>
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <MenuItem onClick={handleToggle} selected={isDark || isLight}>
          <ListItemIcon>
            {resolvedDark ? <Brightness7Icon /> : <Brightness4Icon />}
          </ListItemIcon>
          <ListItemText>
            {resolvedDark ? translate('theme.light') : translate('theme.dark')}
          </ListItemText>
        </MenuItem>
        <MenuItem onClick={handleAuto} selected={isAuto}>
          <ListItemIcon>
            <Brightness4Icon fontSize="small" />
          </ListItemIcon>
          <ListItemText>{translate('theme.auto')}</ListItemText>
        </MenuItem>
      </Menu>
    </>
  )
}
