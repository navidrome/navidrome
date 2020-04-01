import React, { forwardRef } from 'react'
import { MenuItemLink, useTranslate } from 'react-admin'
import { makeStyles } from '@material-ui/core'
import TuneIcon from '@material-ui/icons/Tune'

const useStyles = makeStyles((theme) => ({
  menuItem: {
    color: theme.palette.text.secondary
  }
}))

const PersonalMenu = forwardRef(({ onClick, open, dense }, ref) => {
  const translate = useTranslate()
  const classes = useStyles()
  return (
    <MenuItemLink
      ref={ref}
      to="/personal"
      primaryText={translate('menu.personal')}
      leftIcon={<TuneIcon />}
      onClick={onClick}
      className={classes.menuItem}
      sidebarIsOpen={open}
      dense={dense}
    />
  )
})

export default PersonalMenu
