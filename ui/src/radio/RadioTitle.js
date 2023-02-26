import React from 'react'
import clsx from 'clsx'

import useStyle from '../audioplayer/styles'
import { Link, useTranslate } from 'react-admin'
import ErrorOutline from '@material-ui/icons/ErrorOutline'
import { Button, makeStyles } from '@material-ui/core'

const useStyles = makeStyles(() => ({
  button: {
    textTransform: 'none',
  },
}))

const RadioTitle = React.memo(
  ({ homePageUrl, id, isMobile, name, metadata }) => {
    const classes = useStyle()
    const styles = useStyles()
    const translate = useTranslate()
    const className = classes.audioTitle

    return (
      <Link to={`/radio/${id}/show`} className={className}>
        <span>
          <span className={clsx(classes.songTitle, 'songTitle')}>
            {metadata.title}
          </span>
        </span>
        {isMobile ? (
          <>
            <span className={classes.songInfo}>
              <span className="songArtist">{metadata.artist}</span>
            </span>
            <span className={clsx(classes.songInfo, classes.songAlbum)}>
              <span className="songAlbum">{name}</span>
            </span>
          </>
        ) : (
          <span className={classes.songInfo}>
            {metadata.title && !metadata.artist ? (
              <span className="songArtist">
                <Button
                  variant="outlined"
                  startIcon={<ErrorOutline />}
                  size="small"
                  className={styles.button}
                >
                  {translate('resources.radio.message.noArtist')}
                </Button>
              </span>
            ) : (
              <>
                <span className="songArtist">{metadata.artist}</span> -{' '}
                <span className="songAlbum">{name}</span>
              </>
            )}
          </span>
        )}
      </Link>
    )
  }
)

export default RadioTitle
