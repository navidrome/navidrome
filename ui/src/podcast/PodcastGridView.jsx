import React from 'react'
import {
  GridList,
  GridListTile,
  GridListTileBar,
  Typography,
  useMediaQuery,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import MicIcon from '@material-ui/icons/Mic'
import { useListContext, linkToRecord } from 'react-admin'
import { Link } from 'react-router-dom'
import withWidth from '@material-ui/core/withWidth'

const useStyles = makeStyles((theme) => ({
  root: { margin: theme.spacing(1) },
  tileContainer: { cursor: 'pointer' },
  link: { display: 'block', textDecoration: 'none', color: 'inherit' },
  cover: { width: '100%', display: 'block', objectFit: 'cover' },
  placeholder: {
    width: '100%',
    paddingBottom: '100%',
    position: 'relative',
    backgroundColor: theme.palette.grey[300],
  },
  placeholderIcon: {
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
  },
  tileBar: {
    background: 'linear-gradient(to top, rgba(0,0,0,0.6) 0%, rgba(0,0,0,0) 100%)',
  },
  title: {
    fontSize: '0.85rem',
    fontWeight: 500,
    marginTop: theme.spacing(0.5),
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
}))

const getColsForWidth = (width) => {
  if (width === 'xs') return 2
  if (width === 'sm') return 3
  if (width === 'md') return 4
  if (width === 'lg') return 5
  return 6
}

const PodcastGridView = ({ width, ...props }) => {
  const classes = useStyles()
  const { ids, data, basePath } = useListContext(props)

  if (!ids || !data) return null

  return (
    <div className={classes.root}>
      <GridList cellHeight="auto" cols={getColsForWidth(width)} spacing={16}>
        {ids.map((id) => {
          const record = data[id]
          if (!record) return null
          return (
            <GridListTile key={id}>
              <div className={classes.tileContainer}>
                <Link className={classes.link} to={linkToRecord(basePath, id, 'show')}>
                  {record.imageUrl ? (
                    <img src={record.imageUrl} alt={record.title} className={classes.cover} />
                  ) : (
                    <div className={classes.placeholder}>
                      <MicIcon className={classes.placeholderIcon} style={{ fontSize: 48, color: '#888' }} />
                    </div>
                  )}
                  <GridListTileBar className={classes.tileBar} title="" />
                </Link>
                <Link className={classes.link} to={linkToRecord(basePath, id, 'show')}>
                  <Typography className={classes.title}>{record.title}</Typography>
                </Link>
              </div>
            </GridListTile>
          )
        })}
      </GridList>
    </div>
  )
}

export default withWidth()(PodcastGridView)
