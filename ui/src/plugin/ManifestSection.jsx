import React from 'react'
import {
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Typography,
  Box,
} from '@material-ui/core'
import { MdExpandMore } from 'react-icons/md'

export const ManifestSection = ({ manifestJson, classes, translate }) => (
  <Accordion className={classes.section}>
    <AccordionSummary expandIcon={<MdExpandMore />}>
      <Typography variant="h6">
        {translate('resources.plugin.sections.manifest')}
      </Typography>
    </AccordionSummary>
    <AccordionDetails>
      <Box className={classes.manifestBox} width="100%">
        {manifestJson}
      </Box>
    </AccordionDetails>
  </Accordion>
)
