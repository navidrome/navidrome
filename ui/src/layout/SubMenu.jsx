import React, { Fragment } from 'react'
import { useDispatch } from 'react-redux'
import ExpandMore from '@material-ui/icons/ExpandMore'
import ArrowRightOutlined from '@material-ui/icons/ArrowRightOutlined'
import List from '@material-ui/core/List'
import MenuItem from '@material-ui/core/MenuItem'
import ListItemIcon from '@material-ui/core/ListItemIcon'
import Typography from '@material-ui/core/Typography'
import Collapse from '@material-ui/core/Collapse'
import Tooltip from '@material-ui/core/Tooltip'
import { makeStyles } from '@material-ui/core/styles'
import { setSidebarVisibility, useTranslate } from 'react-admin'
import { IconButton, useMediaQuery } from '@material-ui/core'

const useStyles = makeStyles(
  (theme) => ({
    icon: { minWidth: theme.spacing(5) },
    sidebarIsOpen: {
      '& a': {
        transition: 'padding-left 195ms cubic-bezier(0.4, 0, 0.6, 1) 0ms',
        paddingLeft: theme.spacing(4),
      },
    },
    sidebarIsClosed: {
      '& a': {
        transition: 'padding-left 195ms cubic-bezier(0.4, 0, 0.6, 1) 0ms',
        paddingLeft: theme.spacing(2),
      },
    },
    actionIcon: {
      opacity: 0,
    },
    menuHeader: {
      width: '100%',
    },
    headerWrapper: {
      display: 'flex',
      '&:hover $actionIcon': {
        opacity: 1,
      },
    },
  }),
  {
    name: 'NDSubMenu',
  },
)

const SubMenu = ({
  handleToggle,
  sidebarIsOpen,
  isOpen,
  name,
  icon,
  children,
  dense,
  onAction,
  actionIcon,
}) => {
  const translate = useTranslate()
  const classes = useStyles()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('sm'))
  const isSmall = useMediaQuery((theme) => theme.breakpoints.down('sm'))
  const dispatch = useDispatch()

  const handleOnClick = (e) => {
    e.stopPropagation()
    onAction(e)
    if (isSmall) {
      dispatch(setSidebarVisibility(false))
    }
  }

  const header = (
    <div className={classes.headerWrapper}>
      <MenuItem
        dense={dense}
        button
        className={classes.menuHeader}
        onClick={handleToggle}
      >
        <ListItemIcon className={classes.icon}>
          {isOpen ? <ExpandMore /> : icon}
        </ListItemIcon>
        <Typography variant="inherit" color="textSecondary">
          {translate(name)}
        </Typography>
        {onAction && sidebarIsOpen && (
          <IconButton
            size={'small'}
            className={isDesktop ? classes.actionIcon : null}
            onClick={handleOnClick}
          >
            {actionIcon}
          </IconButton>
        )}
      </MenuItem>
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
      </Collapse>
    </Fragment>
  )
}

SubMenu.defaultProps = {
  action: null,
  actionIcon: <ArrowRightOutlined fontSize={'small'} />,
}

export default SubMenu
