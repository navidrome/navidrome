import * as React from 'react'
import { Children, cloneElement, isValidElement, useState } from 'react'
import PropTypes from 'prop-types'
import { useTranslate, useGetIdentity } from 'react-admin'
import {
  Tooltip,
  IconButton,
  Popover,
  MenuList,
  Button,
  Avatar,
  Card,
  CardContent,
  Divider,
  Typography,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import AccountCircle from '@material-ui/icons/AccountCircle'

const useStyles = makeStyles((theme) => ({
  user: {},
  userButton: {
    textTransform: 'none',
  },
  avatar: {
    width: theme.spacing(4),
    height: theme.spacing(4),
  },
  username: {
    marginTop: '-0.5em',
  },
}))

const UserMenu = (props) => {
  const [anchorEl, setAnchorEl] = useState(null)
  const translate = useTranslate()
  const { loaded, identity } = useGetIdentity()
  const classes = useStyles(props)

  const { children, label, icon, logout } = props
  if (!logout && !children) return null
  const open = Boolean(anchorEl)

  const handleMenu = (event) => setAnchorEl(event.currentTarget)
  const handleClose = () => setAnchorEl(null)

  return (
    <div className={classes.user}>
      <Tooltip title={label && translate(label, { _: label })}>
        <IconButton
          aria-label={label && translate(label, { _: label })}
          aria-owns={open ? 'menu-appbar' : null}
          aria-haspopup={true}
          color="inherit"
          onClick={handleMenu}
        >
          {loaded && identity.avatar ? (
            <Avatar
              className={classes.avatar}
              src={identity.avatar}
              alt={identity.fullName}
            />
          ) : (
            icon
          )}
        </IconButton>
      </Tooltip>
      <Popover
        id="menu-appbar"
        anchorEl={anchorEl}
        anchorOrigin={{
          vertical: 'bottom',
          horizontal: 'right',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'right',
        }}
        open={open}
        onClose={handleClose}
      >
        <MenuList>
          {loaded && (
            <Card elevation={0} className={classes.username}>
              <CardContent>
                <Typography variant={'button'}>{identity.fullName}</Typography>
              </CardContent>
              <Divider />
            </Card>
          )}
          {Children.map(children, (menuItem) =>
            isValidElement(menuItem)
              ? cloneElement(menuItem, {
                  onClick: handleClose,
                })
              : null
          )}
          {logout}
        </MenuList>
      </Popover>
    </div>
  )
}

UserMenu.propTypes = {
  children: PropTypes.node,
  label: PropTypes.string.isRequired,
  logout: PropTypes.element,
}

UserMenu.defaultProps = {
  label: 'ra.auth.user_menu',
  icon: <AccountCircle />,
}

export default UserMenu
