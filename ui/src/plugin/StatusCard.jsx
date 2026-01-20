import React from 'react'
import PropTypes from 'prop-types'
import { Card, CardContent, Typography } from '@material-ui/core'
import ToggleEnabledSwitch from './ToggleEnabledSwitch'

export const StatusCard = ({ classes, translate, manifest }) => {
  return (
    <Card className={classes.section}>
      <CardContent>
        <Typography variant="h6" className={classes.sectionTitle}>
          {translate('resources.plugin.sections.status')}
        </Typography>
        <ToggleEnabledSwitch showLabel size="medium" manifest={manifest} />
      </CardContent>
    </Card>
  )
}

StatusCard.propTypes = {
  classes: PropTypes.object.isRequired,
  translate: PropTypes.func.isRequired,
  manifest: PropTypes.object,
}
