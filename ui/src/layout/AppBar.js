import React, { forwardRef } from 'react'
import { AppBar as RAAppBar, UserMenu, useTranslate } from 'react-admin'
import { makeStyles, MenuItem, ListItemIcon } from '@material-ui/core'
import InfoIcon from '@material-ui/icons/Info'
import AboutDialog from './AboutDialog'

const useStyles = makeStyles((theme) => ({
  root: {
    color: theme.palette.text.secondary,
  },
  active: {
    color: theme.palette.text.primary,
  },
  icon: { minWidth: theme.spacing(5) },
}))

const AboutMenuItem = forwardRef(({ onClick, ...rest }, ref) => {
  const classes = useStyles(rest)
  const translate = useTranslate()
  const [open, setOpen] = React.useState(false)

  const handleOpen = () => {
    setOpen(true)
  }
  const handleClose = () => {
    onClick && onClick()
    setOpen(false)
  }
  const label = translate('menu.about')
  return (
    <>
      <MenuItem
        ref={ref}
        onClick={handleOpen}
        className={classes.root}
        activeClassName={classes.active}
      >
        <ListItemIcon className={classes.icon}>
          <InfoIcon titleAccess={label} />
        </ListItemIcon>
        {label}
      </MenuItem>
      <AboutDialog onClose={handleClose} open={open} />
    </>
  )
})

const CustomUserMenu = (props) => (
  <UserMenu {...props}>
    <AboutMenuItem />
  </UserMenu>
)

const AppBar = (props) => <RAAppBar {...props} userMenu={<CustomUserMenu />} />

export default AppBar
