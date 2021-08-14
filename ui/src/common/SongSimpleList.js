import React from 'react'
import PropTypes from 'prop-types'
import ListItem from '@material-ui/core/ListItem'
import ListItemIcon from '@material-ui/core/ListItemIcon'
import ListItemSecondaryAction from '@material-ui/core/ListItemSecondaryAction'
import ListItemText from '@material-ui/core/ListItemText'
import { makeStyles } from '@material-ui/core/styles'
import { DurationField, SongContextMenu, RatingField } from './index'
import { setTrack } from '../actions'
import { useDispatch } from 'react-redux'
import VirtualList from '../infiniteScroll/VirtualList'
import config from '../config'

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
  { name: 'RaSongSimpleList' }
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
      <VirtualList
        className={className}
        itemHeight={100}
        renderItem={(record) =>
          record && (
            <span key={record.id} onClick={() => dispatch(setTrack(record))}>
              <ListItem className={classes.listItem} button={true}>
                <ListItemText
                  primary={<div className={classes.title}>{record.title}</div>}
                  secondary={
                    <>
                      <span className={classes.secondary}>
                        <span className={classes.artist}>{record.artist}</span>
                        <span className={classes.timeStamp}>
                          <DurationField record={record} source={'duration'} />
                        </span>
                      </span>
                      {config.enableStarRating && (
                        <RatingField
                          record={record}
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
                    <SongContextMenu record={record} visible={true} />
                  </ListItemIcon>
                </ListItemSecondaryAction>
              </ListItem>
            </span>
          )
        }
      />
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
