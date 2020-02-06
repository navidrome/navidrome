import React, { forwardRef } from 'react';
import { AppBar as RAAppBar, UserMenu, MenuItemLink } from 'react-admin'
import InfoIcon from '@material-ui/icons/Info';

const ConfigurationMenu = forwardRef(({ onClick }, ref) => (
  <MenuItemLink
    ref={ref}
    to=""
    primaryText={"Version " + localStorage.getItem("version") }
    leftIcon={<InfoIcon />}
    onClick={onClick}
  />
))

const CustomUserMenu = (props) => (
  <UserMenu {...props}>
    <ConfigurationMenu />
  </UserMenu>
)

const AppBar = (props) => <RAAppBar {...props} userMenu={<CustomUserMenu />} />

export default AppBar
