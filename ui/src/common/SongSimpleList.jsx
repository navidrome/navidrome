import React from 'react'
import PropTypes from 'prop-types'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemSecondaryAction from '@mui/material/ListItemSecondaryAction'
import ListItemText from '@mui/material/ListItemText'
import { useTheme } from '@mui/material/styles'
import { sanitizeListRestProps } from 'react-admin'
import { DurationField, SongContextMenu, RatingField } from './index'
import { setTrack } from '../actions'
import { useDispatch } from 'react-redux'
import config from '../config'

// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles(
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
      paddingRight: '30px',
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
  { name: 'RaSongSimpleList' },
)

export const SongSimpleList = ({
  basePath,
  className,
  classes: classesOverride,
  data,
  hasBulkActions,
  ids,
  loading,
  onToggleItem,
  selectedIds,
  total,
  ...rest
}) => {
  const dispatch = useDispatch()
  const classes = useStyles({ classes: classesOverride })
  return (
    (loading || total > 0) && (
      <List className={className} {...sanitizeListRestProps(rest)}>
        {ids.map(
          (id) =>
            data[id] && (
              <span key={id} onClick={() => dispatch(setTrack(data[id]))}>
                <ListItem className={classes.listItem} button={true}>
                  <ListItemText
                    primary={
                      <div className={classes.title}>{data[id].title}</div>
                    }
                    secondary={
                      <>
                        <span className={classes.secondary}>
                          <span className={classes.artist}>
                            {data[id].artist}
                          </span>
                          <span className={classes.timeStamp}>
                            <DurationField
                              record={data[id]}
                              source={'duration'}
                            />
                          </span>
                        </span>
                        {config.enableStarRating && (
                          <RatingField
                            record={data[id]}
                            source={'rating'}
                            resource={'song'}
                            size={'small'}
                          />
                        )}
                      </>
                    }
                  />
                  <ListItemSecondaryAction className={classes.rightIcon}>
                    <ListItemIcon>
                      <SongContextMenu record={data[id]} visible={true} />
                    </ListItemIcon>
                  </ListItemSecondaryAction>
                </ListItem>
              </span>
            ),
        )}
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
  onToggleItem: PropTypes.func,
  selectedIds: PropTypes.arrayOf(PropTypes.any).isRequired,
}

SongSimpleList.defaultProps = {
  hasBulkActions: false,
  selectedIds: [],
}
