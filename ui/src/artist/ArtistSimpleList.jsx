import React from 'react'
import PropTypes from 'prop-types'
import List from '@material-ui/core/List'
import ListItem from '@material-ui/core/ListItem'
import ListItemIcon from '@material-ui/core/ListItemIcon'
import ListItemSecondaryAction from '@material-ui/core/ListItemSecondaryAction'
import ListItemText from '@material-ui/core/ListItemText'
import { makeStyles } from '@material-ui/core/styles'
import { sanitizeListRestProps } from 'react-admin'
import { ArtistContextMenu, RatingField } from '../common'
import config from '../config'

const useStyles = makeStyles(
  {
    listItem: {
      padding: '10px',
    },
    title: {
      paddingRight: '10px',
      width: '80%',
    },
    rightIcon: {
      top: '26px',
    },
  },
  { name: 'RaArtistSimpleList' },
)

const ArtistSimpleList = ({
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
      <List className={className} {...sanitizeListRestProps(rest)}>
        {ids.map(
          (id) =>
            data[id] && (
              <span key={id} onClick={() => linkType(id)}>
                <ListItem className={classes.listItem} button={true}>
                  <ListItemText
                    primary={
                      <>
                        <div className={classes.title}>{data[id].name}</div>
                        {config.enableStarRating && (
                          <RatingField
                            record={data[id]}
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
                      <ArtistContextMenu record={data[id]} />
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

export default ArtistSimpleList
