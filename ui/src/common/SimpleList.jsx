import React from 'react'
import PropTypes from 'prop-types'
import Avatar from '@mui/material/Avatar'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemAvatar from '@mui/material/ListItemAvatar'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemSecondaryAction from '@mui/material/ListItemSecondaryAction'
import ListItemText from '@mui/material/ListItemText'
import { useTheme } from '@mui/material/styles'
import { Link } from 'react-router-dom'
import { linkToRecord, sanitizeListRestProps } from 'react-admin'

// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles(
  {
    link: {
      textDecoration: 'none',
      color: 'inherit',
    },
    tertiary: { float: 'right', opacity: 0.541176 },
  },
  { name: 'RaSimpleList' },
)

const LinkOrNot = ({
  classes: classesOverride,
  linkType,
  basePath,
  id,
  record,
  children,
}) => {
  const classes = useStyles({ classes: classesOverride })
  return linkType === 'edit' || linkType === true ? (
    <Link to={linkToRecord(basePath, id)} className={classes.link}>
      {children}
    </Link>
  ) : linkType === 'show' ? (
    <Link to={`${linkToRecord(basePath, id)}/show`} className={classes.link}>
      {children}
    </Link>
  ) : typeof linkType === 'function' ? (
    <span onClick={() => linkType(id, basePath, record)}>{children}</span>
  ) : (
    <span>{children}</span>
  )
}

export const SimpleList = ({
  basePath,
  className,
  classes: classesOverride,
  data,
  hasBulkActions,
  ids,
  loading,
  leftAvatar,
  leftIcon,
  linkType,
  onToggleItem,
  primaryText,
  rightAvatar,
  rightIcon,
  secondaryText,
  selectedIds,
  tertiaryText,
  total,
  ...rest
}) => {
  const classes = useStyles({ classes: classesOverride })
  return (
    (loading || total > 0) && (
      <List className={className} {...sanitizeListRestProps(rest)}>
        {ids.map((id) => (
          <LinkOrNot
            linkType={linkType}
            basePath={basePath}
            id={id}
            key={id}
            record={data[id]}
          >
            <ListItem button={!!linkType}>
              {leftIcon && (
                <ListItemIcon>{leftIcon(data[id], id)}</ListItemIcon>
              )}
              {leftAvatar && (
                <ListItemAvatar>
                  <Avatar>{leftAvatar(data[id], id)}</Avatar>
                </ListItemAvatar>
              )}
              <ListItemText
                primary={
                  <div>
                    {primaryText(data[id], id)}
                    {tertiaryText && (
                      <span className={classes.tertiary}>
                        {tertiaryText(data[id], id)}
                      </span>
                    )}
                  </div>
                }
                secondary={secondaryText && secondaryText(data[id], id)}
              />
              {(rightAvatar || rightIcon) && (
                <ListItemSecondaryAction>
                  {rightAvatar && <Avatar>{rightAvatar(data[id], id)}</Avatar>}
                  {rightIcon && (
                    <ListItemIcon>{rightIcon(data[id], id)}</ListItemIcon>
                  )}
                </ListItemSecondaryAction>
              )}
            </ListItem>
          </LinkOrNot>
        ))}
      </List>
    )
  )
}

SimpleList.propTypes = {
  basePath: PropTypes.string,
  className: PropTypes.string,
  classes: PropTypes.object,
  data: PropTypes.object,
  hasBulkActions: PropTypes.bool.isRequired,
  ids: PropTypes.array,
  leftAvatar: PropTypes.func,
  leftIcon: PropTypes.func,
  linkType: PropTypes.oneOfType([
    PropTypes.string,
    PropTypes.bool,
    PropTypes.func,
  ]).isRequired,
  onToggleItem: PropTypes.func,
  primaryText: PropTypes.func,
  rightAvatar: PropTypes.func,
  rightIcon: PropTypes.func,
  secondaryText: PropTypes.func,
  selectedIds: PropTypes.arrayOf(PropTypes.any).isRequired,
  tertiaryText: PropTypes.func,
}

SimpleList.defaultProps = {
  linkType: 'edit',
  hasBulkActions: false,
  selectedIds: [],
}
