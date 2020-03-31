import React, { forwardRef } from 'react'
import {
  AppBar as RAAppBar,
  UserMenu,
  MenuItemLink,
  useTranslate
} from 'react-admin'
import InfoIcon from '@material-ui/icons/Info'
import TuneIcon from '@material-ui/icons/Tune'

const ConfigurationMenu = forwardRef(({ onClick }, ref) => {
  const translate = useTranslate()
  return (
    <MenuItemLink
      ref={ref}
      to="/configuration"
      primaryText={translate('menu.configuration')}
      leftIcon={<TuneIcon />}
      onClick={onClick}
    />
  )
})

const VersionMenu = forwardRef(({ onClick }, ref) => (
  <MenuItemLink
    ref={ref}
    to=""
    primaryText={'Version ' + localStorage.getItem('version')}
    leftIcon={<InfoIcon />}
    onClick={onClick}
  />
))

const CustomUserMenu = (props) => (
  <UserMenu {...props}>
    <ConfigurationMenu />
    <VersionMenu />
  </UserMenu>
)

const AppBar = (props) => <RAAppBar {...props} userMenu={<CustomUserMenu />} />

export default AppBar
