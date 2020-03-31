import React, { forwardRef } from 'react'
import {
  AppBar as RAAppBar,
  UserMenu,
  MenuItemLink,
  useTranslate
} from 'react-admin'
import { makeStyles } from '@material-ui/core'
import InfoIcon from '@material-ui/icons/Info'
import TuneIcon from '@material-ui/icons/Tune'

const useStyles = makeStyles((theme) => ({
  menuItem: {
    color: theme.palette.text.secondary
  }
}))

const ConfigurationMenu = forwardRef(({ onClick }, ref) => {
  const translate = useTranslate()
  const classes = useStyles()
  return (
    <MenuItemLink
      ref={ref}
      to="/configuration"
      primaryText={translate('menu.configuration')}
      leftIcon={<TuneIcon />}
      onClick={onClick}
      className={classes.menuItem}
    />
  )
})

const VersionMenu = forwardRef(({ onClick }, ref) => {
  const classes = useStyles()
  return (
    <MenuItemLink
      ref={ref}
      to=""
      primaryText={'Version ' + localStorage.getItem('version')}
      leftIcon={<InfoIcon />}
      onClick={onClick}
      className={classes.menuItem}
    />
  )
})

const CustomUserMenu = (props) => (
  <UserMenu {...props}>
    <ConfigurationMenu />
    <VersionMenu />
  </UserMenu>
)

const AppBar = (props) => <RAAppBar {...props} userMenu={<CustomUserMenu />} />

export default AppBar
