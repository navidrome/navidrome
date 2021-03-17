import React from 'react'
import PropTypes from 'prop-types'
import List from '@material-ui/core/List'
import ListItem from '@material-ui/core/ListItem'
import ListItemIcon from '@material-ui/core/ListItemIcon'
import ListItemSecondaryAction from '@material-ui/core/ListItemSecondaryAction'
import ListItemText from '@material-ui/core/ListItemText'
import Typography from '@material-ui/core/Typography'
import { makeStyles } from '@material-ui/core/styles'
import { Link } from 'react-router-dom'
import { linkToRecord, sanitizeListRestProps } from 'ra-core'

const useStyles = makeStyles(
  {
    link: {
      textDecoration: 'none',
      color: 'inherit',
    },
    listItem: {
      padding: '10px',
    },
    title: {
      paddingRight: '10px',
      width: '80%',
    },
    secondary: {
      marginTop: '-3px',
      width: '96%',
      display: 'flex',
      alignItems: 'flex-start',
      justifyContent: 'space-between',
    },
    artist: {
      paddingRight: '20px',
    },
    timeStamp: {
      float: 'right',
      color: '#fff',
      fontWeight: '200',
      opacity: 0.6,
      fontSize: '12px',
      padding: '2px',
    },
    rightIcon: {
      top: '26px',
    },
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

export const SongSimpleList = ({
  title,
  author,
  songTime,
  basePath,
  className,
  classes: classesOverride,
  data,
  hasBulkActions,
  ids,
  loading,
  leftIcon,
  linkType,
  onToggleItem,
  rightIcon,
  selectedIds,
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
            <ListItem className={classes.listItem} button={!!linkType}>
              {leftIcon && (
                <ListItemIcon>{leftIcon(data[id], id)}</ListItemIcon>
              )}
              <ListItemText
                primary={
                  <div className={classes.title}>{title(data[id], id)}</div>
                }
                secondary={
                  <div className={classes.secondary}>
                    <div className={classes.artist}>
                      {author && author(data[id], id)}
                    </div>
                    {songTime && (
                      <span className={classes.timeStamp}>
                        {songTime(data[id], id)}
                      </span>
                    )}
                  </div>
                }
              />
              <ListItemSecondaryAction className={classes.rightIcon}>
                {rightIcon && (
                  <ListItemIcon>{rightIcon(data[id], id)}</ListItemIcon>
                )}
              </ListItemSecondaryAction>
            </ListItem>
          </LinkOrNot>
        ))}
      </List>
    )
  )
}

SongSimpleList.propTypes = {
  basePath: PropTypes.string,
  className: PropTypes.string,
  classes: PropTypes.object,
  data: PropTypes.object,
  hasBulkActions: PropTypes.bool.isRequired,
  ids: PropTypes.array,
  leftIcon: PropTypes.func,
  linkType: PropTypes.oneOfType([
    PropTypes.string,
    PropTypes.bool,
    PropTypes.func,
  ]).isRequired,
  onToggleItem: PropTypes.func,
  title: PropTypes.func,
  rightIcon: PropTypes.func,
  author: PropTypes.func,
  selectedIds: PropTypes.arrayOf(PropTypes.any).isRequired,
  songTime: PropTypes.func,
}

SongSimpleList.defaultProps = {
  linkType: 'edit',
  hasBulkActions: false,
  selectedIds: [],
}

