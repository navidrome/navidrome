import React, { forwardRef } from 'react'
import { MenuItemLink, useTranslate } from 'react-admin'
import { MdTune } from 'react-icons/md'
import { makeStyles } from '@material-ui/core'

const useStyles = makeStyles((theme) => ({
  menuItem: {
    color: theme.palette.text.secondary,
  },
}))

const PersonalMenu = forwardRef(({ onClick, sidebarIsOpen, dense }, ref) => {
  const translate = useTranslate()
  const classes = useStyles()
  return (
    <MenuItemLink
      ref={ref}
      to="/personal"
      primaryText={translate('menu.personal.name')}
      leftIcon={<MdTune size={24} />}
      onClick={onClick}
      className={classes.menuItem}
      sidebarIsOpen={sidebarIsOpen}
      dense={dense}
    />
  )
})

PersonalMenu.displayName = 'PersonalMenu'

export default PersonalMenu
