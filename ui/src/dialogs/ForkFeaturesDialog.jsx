import React from 'react'
import PropTypes from 'prop-types'
import Dialog from '@material-ui/core/Dialog'
import Accordion from '@material-ui/core/Accordion'
import AccordionSummary from '@material-ui/core/AccordionSummary'
import AccordionDetails from '@material-ui/core/AccordionDetails'
import Typography from '@material-ui/core/Typography'
import { MdExpandMore } from 'react-icons/md'
import { useTranslate } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { DialogTitle } from './DialogTitle'
import { DialogContent } from './DialogContent'

const useStyles = makeStyles((theme) => ({
  intro: {
    marginBottom: theme.spacing(2),
  },
  overview: {
    marginBottom: theme.spacing(1),
  },
  howToLabel: {
    fontWeight: 600,
    marginBottom: theme.spacing(0.5),
  },
}))

const FEATURE_KEYS = [
  'podcasts',
  'folders',
  'tagging',
  'skipSongs',
  'scrobbleAttribution',
  'genreExploration',
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
        {FEATURE_KEYS.map((key) => (
          <Accordion key={key}>
            <AccordionSummary expandIcon={<MdExpandMore />}>
              <Typography variant="subtitle1">
                {translate(`forkFeatures.${key}.title`)}
              </Typography>
            </AccordionSummary>
            <AccordionDetails>
              <div>
                <Typography variant="body2" className={classes.overview}>
                  {translate(`forkFeatures.${key}.overview`)}
                </Typography>
                <Typography variant="body2" className={classes.howToLabel}>
                  {translate('forkFeatures.howToLabel')}
                </Typography>
                <Typography variant="body2">
                  {translate(`forkFeatures.${key}.howTo`)}
                </Typography>
              </div>
            </AccordionDetails>
          </Accordion>
        ))}
      </DialogContent>
    </Dialog>
  )
}

ForkFeaturesDialog.propTypes = {
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
}
