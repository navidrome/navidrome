import React from 'react'
import PropTypes from 'prop-types'
import ListItem from '@material-ui/core/ListItem'
import ListItemIcon from '@material-ui/core/ListItemIcon'
import ListItemSecondaryAction from '@material-ui/core/ListItemSecondaryAction'
import ListItemText from '@material-ui/core/ListItemText'
import { makeStyles } from '@material-ui/core/styles'
import { ArtistContextMenu, RatingField } from './index'
import VirtualList from '../infiniteScroll/VirtualList'
import config from '../config'

const useStyles = makeStyles(
  {
    listItem: {
      padding: '10px',
      listStyleType: 'none',
    },
    title: {
      paddingRight: '10px',
      width: '80%',
    },
    rightIcon: {
      top: '26px',
    },
  },
  { name: 'RaArtistSimpleList' }
)

export const ArtistSimpleList = ({
  linkType,
  className,
  classes: classesOverride,
  data,
  hasBulkActions,
  ids,
  loading,
  selectedIds,
  total,
  ...rest
}) => {
  const classes = useStyles({ classes: classesOverride })
  return (
    (loading || total > 0) && (
      <VirtualList
        className={className}
        itemHeight={75}
        renderItem={(record) =>
          record && (
            <span key={record.id} onClick={() => linkType(record.id)}>
              <ListItem className={classes.listItem} button={true}>
                <ListItemText
                  primary={
                    <>
                      <div className={classes.title}>{record.name}</div>
                      {config.enableStarRating && (
                        <RatingField
                          record={record}
                          source={'rating'}
                          resource={'artist'}
                          size={'small'}
                        />
                      )}
                    </>
                  }
                />
                <ListItemSecondaryAction className={classes.rightIcon}>
                  <ListItemIcon>
                    <ArtistContextMenu record={record} />
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

ArtistSimpleList.propTypes = {
  className: PropTypes.string,
  classes: PropTypes.object,
  data: PropTypes.object,
  hasBulkActions: PropTypes.bool.isRequired,
  ids: PropTypes.array,
  selectedIds: PropTypes.arrayOf(PropTypes.any).isRequired,
}

ArtistSimpleList.defaultProps = {
  hasBulkActions: false,
  selectedIds: [],
}
