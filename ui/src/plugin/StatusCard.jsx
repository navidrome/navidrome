import React from 'react'
import { Card, CardContent, Typography } from '@material-ui/core'
import ToggleEnabledSwitch from './ToggleEnabledSwitch'

export const StatusCard = ({ classes, translate }) => {
  return (
    <Card className={classes.section}>
      <CardContent>
        <Typography variant="h6" className={classes.sectionTitle}>
          {translate('resources.plugin.sections.status')}
        </Typography>
        <ToggleEnabledSwitch showLabel size="medium" />
      </CardContent>
    </Card>
  )
}
