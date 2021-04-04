import React, { Fragment } from 'react'
import { useHistory } from 'react-router-dom'
import ExpandMore from '@material-ui/icons/ExpandMore'
import List from '@material-ui/core/List'
import MenuItem from '@material-ui/core/MenuItem'
import ListItemIcon from '@material-ui/core/ListItemIcon'
import Typography from '@material-ui/core/Typography'
import Divider from '@material-ui/core/Divider'
import Collapse from '@material-ui/core/Collapse'
import Tooltip from '@material-ui/core/Tooltip'
import { makeStyles } from '@material-ui/core/styles'
import { useMediaQuery } from '@material-ui/core'
import ArrowRightOutlinedIcon from '@material-ui/icons/ArrowRightOutlined'
import { MenuItemLink } from 'react-admin'

const useStyles = makeStyles((theme) => ({
  icon: { minWidth: theme.spacing(5) },
  sidebarIsOpen: {
    paddingLeft: 25,
    transition: 'padding-left 195ms cubic-bezier(0.4, 0, 0.6, 1) 0ms',
  },
  sidebarIsClosed: {
    paddingLeft: 0,
    transition: 'padding-left 195ms cubic-bezier(0.4, 0, 0.6, 1) 0ms',
  },
  secondaryIcon: {
    opacity: 0,
  },
  menuHeader: {
    width: '100%',
  },
  headerWrapper: {
    display: 'flex',
    '&:hover $secondaryIcon': {
      opacity: 1,
    },
  },
}))

const SubMenu = ({
  handleToggle,
  sidebarIsOpen,
  isOpen,
  name,
  icon,
  children,
  dense,
  secondaryLink,
  secondaryAction,
}) => {
  const classes = useStyles()
  const history = useHistory()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('sm'))

  if (secondaryLink) {
    if (isOpen && sidebarIsOpen) {
      icon = <ExpandMore />
    }
  } else {
    if (isOpen) {
      icon = <ExpandMore />
    }
  }

  const handleClick = () => {
    if (secondaryLink && !sidebarIsOpen) {
      history.push(secondaryLink)
    } else {
      handleToggle()
    }
  }

  const header = (
    <div className={classes.headerWrapper}>
      <MenuItem
        dense={dense}
        button
        className={classes.menuHeader}
        onClick={handleClick}
      >
        <ListItemIcon className={classes.icon}>{icon}</ListItemIcon>
        <Typography variant="inherit" color="textSecondary">
          {name}
        </Typography>
      </MenuItem>
      {secondaryLink && sidebarIsOpen ? (
        <MenuItemLink
          className={isDesktop ? classes.secondaryIcon : null}
          to={secondaryLink}
          primaryText={<ArrowRightOutlinedIcon fontSize="small" />}
          onClick={secondaryAction}
          tooltipProps={{
            disableHoverListener: true,
          }}
        />
      ) : null}
    </div>
  )

  return (
    <Fragment>
      {sidebarIsOpen || isOpen ? (
        header
      ) : (
        <Tooltip title={name} placement="right">
          {header}
        </Tooltip>
      )}
      <Collapse in={isOpen} timeout="auto" unmountOnExit>
        <List
          dense={dense}
          component="div"
          disablePadding
          className={
            sidebarIsOpen ? classes.sidebarIsOpen : classes.sidebarIsClosed
          }
        >
          {children}
        </List>
        <Divider />
      </Collapse>
    </Fragment>
  )
}

export default SubMenu

SubMenu.defaultProps = {
  secondaryLink: '',
  dense: false,
}
