import React, { Fragment } from 'react'
import ExpandMore from '@material-ui/icons/ExpandMore'
import List from '@material-ui/core/List'
import MenuItem from '@material-ui/core/MenuItem'
import ListItemIcon from '@material-ui/core/ListItemIcon'
import Typography from '@material-ui/core/Typography'
import Divider from '@material-ui/core/Divider'
import Collapse from '@material-ui/core/Collapse'
import Tooltip from '@material-ui/core/Tooltip'
import { makeStyles } from '@material-ui/core/styles'
import { MenuItemLink, useTranslate } from 'react-admin'
import ArrowForwardIcon from '@material-ui/icons/ArrowForward'

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
  menuHeader: {
    width: '100%',
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
  secondaryLink = '',
  secondaryAction,
}) => {
  const translate = useTranslate()
  const classes = useStyles()

  const header = (
    <div style={{ display: 'flex' }}>
      <MenuItem
        dense={dense}
        onClick={handleToggle}
        className={classes.menuHeader}
      >
        <ListItemIcon className={classes.icon}>
          {isOpen ? <ExpandMore /> : icon}
        </ListItemIcon>
        <Typography variant="inherit" color="textSecondary">
          {translate(name)}
        </Typography>
      </MenuItem>
      {secondaryLink ? (
        <MenuItemLink
          to={secondaryLink}
          primaryText={<ArrowForwardIcon />}
          onClick={secondaryAction}
        />
      ) : null}
    </div>
  )

  return (
    <Fragment>
      {sidebarIsOpen || isOpen ? (
        header
      ) : (
        <Tooltip title={translate(name)} placement="right">
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
