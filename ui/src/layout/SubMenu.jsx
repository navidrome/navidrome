import React, { Fragment } from 'react'
import { useDispatch } from 'react-redux'
import ExpandMore from '@mui/icons-material/ExpandMore'
import ArrowRightOutlined from '@mui/icons-material/ArrowRightOutlined'
import List from '@mui/material/List'
import MenuItem from '@mui/material/MenuItem'
import ListItemIcon from '@mui/material/ListItemIcon'
import Typography from '@mui/material/Typography'
import Collapse from '@mui/material/Collapse'
import Tooltip from '@mui/material/Tooltip'
import { useTheme } from '@mui/material/styles'
import { setSidebarVisibility, useTranslate } from 'react-admin'
import { IconButton, useMediaQuery } from '@mui/material'

// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles(
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
    headerText: {
      flexGrow: 1,
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
  onSecondaryAction,
  secondaryActionIcon,
  secondaryActionTitle,
  secondaryActionActive,
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

  const handleSecondaryClick = (e) => {
    e.stopPropagation()
    onSecondaryAction(e)
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
        <Typography
          variant="inherit"
          color="textSecondary"
          className={classes.headerText}
        >
          {translate(name)}
        </Typography>
        {onSecondaryAction && sidebarIsOpen && (
          <IconButton
            size={'small'}
            title={secondaryActionTitle}
            aria-label={secondaryActionTitle}
            className={
              isDesktop && !secondaryActionActive ? classes.actionIcon : null
            }
            onClick={handleSecondaryClick}
          >
            {secondaryActionIcon}
          </IconButton>
        )}
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
