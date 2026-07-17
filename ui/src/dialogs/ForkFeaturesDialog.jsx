import React from 'react'
import PropTypes from 'prop-types'
import Dialog from '@material-ui/core/Dialog'
import List from '@material-ui/core/List'
import ListItem from '@material-ui/core/ListItem'
import ListItemText from '@material-ui/core/ListItemText'
import Link from '@material-ui/core/Link'
import Typography from '@material-ui/core/Typography'
import { useTranslate } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { DialogTitle } from './DialogTitle'
import { DialogContent } from './DialogContent'
import { FORK_README_URL } from '../consts'

const useStyles = makeStyles((theme) => ({
  intro: {
    marginBottom: theme.spacing(2),
  },
}))

const FEATURES = [
  { key: 'podcasts', anchor: 'podcast-support-experimental' },
  { key: 'folders', anchor: 'physical-folder-browsing-experimental' },
  { key: 'tagging', anchor: 'user-defined-song-tagging-experimental' },
  { key: 'skipSongs', anchor: 'skip--auto-pass-disliked-songs-experimental' },
  {
    key: 'scrobbleAttribution',
    anchor: 'enhanced-scrobble-attribution-pulse-integration',
  },
]

export const ForkFeaturesDialog = ({ open, onClose }) => {
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <Dialog
      onClose={onClose}
      aria-labelledby="fork-features-dialog-title"
      open={open}
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="fork-features-dialog-title" onClose={onClose}>
        {translate('forkFeatures.title')}
      </DialogTitle>
      <DialogContent dividers>
        <Typography variant="body2" className={classes.intro}>
          {translate('forkFeatures.intro')}
        </Typography>
        <List>
          {FEATURES.map(({ key, anchor }) => (
            <ListItem key={key} disableGutters>
              <ListItemText
                primary={translate(`forkFeatures.${key}.title`)}
                secondary={
                  <>
                    {translate(`forkFeatures.${key}.description`)}{' '}
                    <Link
                      href={`${FORK_README_URL}${anchor}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {translate('forkFeatures.learnMore')}
                    </Link>
                  </>
                }
              />
            </ListItem>
          ))}
        </List>
      </DialogContent>
    </Dialog>
  )
}

ForkFeaturesDialog.propTypes = {
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
}
