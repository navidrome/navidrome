import React from 'react'
import PropTypes from 'prop-types'
import Avatar from '@material-ui/core/Avatar'
import ListItem from '@material-ui/core/ListItem'
import ListItemAvatar from '@material-ui/core/ListItemAvatar'
import ListItemIcon from '@material-ui/core/ListItemIcon'
import ListItemSecondaryAction from '@material-ui/core/ListItemSecondaryAction'
import ListItemText from '@material-ui/core/ListItemText'
import { makeStyles } from '@material-ui/core/styles'
import { Link } from 'react-router-dom'
import { linkToRecord } from 'react-admin'
import VirtualList from '../infiniteScroll/VirtualList'

const useStyles = makeStyles(
  {
    link: {
      textDecoration: 'none',
      color: 'inherit',
    },
    tertiary: { float: 'right', opacity: 0.541176 },
  },
  { name: 'RaSimpleList' }
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
      <VirtualList
        className={className}
        renderItem={(record) =>
          record && (
            <LinkOrNot
              linkType={linkType}
              basePath={basePath}
              id={record.id}
              key={record.id}
              record={record}
            >
              <ListItem button={!!linkType}>
                {leftIcon && (
                  <ListItemIcon>{leftIcon(record, record.id)}</ListItemIcon>
                )}
                {leftAvatar && (
                  <ListItemAvatar>
                    <Avatar>{leftAvatar(record, record.id)}</Avatar>
                  </ListItemAvatar>
                )}
                <ListItemText
                  primary={
                    <div>
                      {primaryText(record, record.id)}
                      {tertiaryText && (
                        <span className={classes.tertiary}>
                          {tertiaryText(record, record.id)}
                        </span>
                      )}
                    </div>
                  }
                  secondary={secondaryText && secondaryText(record, record.id)}
                />
                {(rightAvatar || rightIcon) && (
                  <ListItemSecondaryAction>
                    {rightAvatar && (
                      <Avatar>{rightAvatar(record, record.id)}</Avatar>
                    )}
                    {rightIcon && (
                      <ListItemIcon>
                        {rightIcon(record, record.id)}
                      </ListItemIcon>
                    )}
                  </ListItemSecondaryAction>
                )}
              </ListItem>
            </LinkOrNot>
          )
        }
      />
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
