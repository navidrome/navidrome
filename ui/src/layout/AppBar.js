import React, { forwardRef } from 'react'
import {
  AppBar as RAAppBar,
  MenuItemLink,
  UserMenu,
  useTranslate,
} from 'react-admin'
import { makeStyles } from '@material-ui/core'
import InfoIcon from '@material-ui/icons/Info'
import config from '../config'

const useStyles = makeStyles((theme) => ({
  menuItem: {
    color: theme.palette.text.secondary,
  },
}))

const VersionMenu = forwardRef((props, ref) => {
  const translate = useTranslate()
  const classes = useStyles()
  return (
    <MenuItemLink
      ref={ref}
      to="#"
      primaryText={translate('menu.version', {
        version: config.version,
      })}
      leftIcon={<InfoIcon />}
      className={classes.menuItem}
      sidebarIsOpen={true}
    />
  )
})

const CustomUserMenu = (props) => (
  <UserMenu {...props}>
    <VersionMenu />
  </UserMenu>
)

const AppBar = (props) => <RAAppBar {...props} userMenu={<CustomUserMenu />} />

export default AppBar
