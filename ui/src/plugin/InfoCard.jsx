import React, { useState } from 'react'
import {
  Card,
  CardContent,
  Typography,
  Grid,
  Box,
  Chip,
  Tooltip,
  Link,
  ClickAwayListener,
} from '@material-ui/core'
import { useTranslate } from 'react-admin'
import { DateField } from '../common'

// Helper component for permission chips with clickable persistent tooltips
const PermissionChip = ({ label, permission, classes }) => {
  const [open, setOpen] = useState(false)
  const translate = useTranslate()

  if (!permission) return null

  const hasHosts = permission.requiredHosts?.length > 0
  const hasTooltip = permission.reason || hasHosts

  const handleClick = () => {
    if (hasTooltip) {
      setOpen((prev) => !prev)
    }
  }

  const handleClose = () => {
    setOpen(false)
  }

  const tooltipContent = (
    <Box className={classes.tooltipContent}>
      {permission.reason && (
        <Typography variant="body2">{permission.reason}</Typography>
      )}
      {hasHosts && (
        <Box mt={permission.reason ? 0.5 : 0}>
          <Typography variant="caption" component="div">
            {translate('resources.plugin.messages.requiredHosts')}:{' '}
            {permission.requiredHosts.map((host, i) => (
              <span key={host}>
                {i > 0 && ', '}
                <code>{host}</code>
              </span>
            ))}
          </Typography>
        </Box>
      )}
    </Box>
  )

  const chip = (
    <Chip
      size="small"
      label={label}
      className={classes.permissionChip}
      onClick={hasTooltip ? handleClick : undefined}
      clickable={hasTooltip}
    />
  )

  if (!hasTooltip) {
    return chip
  }

  return (
    <ClickAwayListener onClickAway={handleClose}>
      <div>
        <Tooltip
          title={tooltipContent}
          arrow
          open={open}
          disableFocusListener
          disableHoverListener
          disableTouchListener
          PopperProps={{
            disablePortal: true,
          }}
        >
          {chip}
        </Tooltip>
      </div>
    </ClickAwayListener>
  )
}

// Info row component for responsive grid
const InfoRow = ({ label, children, classes, isSmall }) => (
  <>
    <Grid item xs={12} sm={3}>
      <Typography
        variant="body2"
        className={classes.infoLabel}
        component={isSmall ? 'div' : 'span'}
      >
        {label}
      </Typography>
    </Grid>
    <Grid item xs={12} sm={9}>
      <Typography variant="body2" component="div">
        {children}
      </Typography>
    </Grid>
  </>
)

// Plugin information card
export const InfoCard = ({ record, manifest, classes, translate, isSmall }) => (
  <Card className={classes.section}>
    <CardContent>
      <Typography variant="h6" className={classes.sectionTitle}>
        {translate('resources.plugin.sections.info')}
      </Typography>
      <Grid container spacing={1} className={classes.infoGrid}>
        <InfoRow
          label={translate('resources.plugin.fields.id')}
          classes={classes}
          isSmall={isSmall}
        >
          {record.id}
        </InfoRow>

        {manifest?.name && (
          <InfoRow
            label={translate('resources.plugin.fields.name')}
            classes={classes}
            isSmall={isSmall}
          >
            {manifest.name}
          </InfoRow>
        )}

        {manifest?.version && (
          <InfoRow
            label={translate('resources.plugin.fields.version')}
            classes={classes}
            isSmall={isSmall}
          >
            {manifest.version}
          </InfoRow>
        )}

        {manifest?.description && (
          <InfoRow
            label={translate('resources.plugin.fields.description')}
            classes={classes}
            isSmall={isSmall}
          >
            {manifest.description}
          </InfoRow>
        )}

        {manifest?.author && (
          <InfoRow
            label={translate('resources.plugin.fields.author')}
            classes={classes}
            isSmall={isSmall}
          >
            {manifest.author}
          </InfoRow>
        )}

        {manifest?.website && (
          <InfoRow
            label={translate('resources.plugin.fields.website')}
            classes={classes}
            isSmall={isSmall}
          >
            <Link
              href={manifest.website}
              target="_blank"
              rel="noopener noreferrer"
            >
              {manifest.website}
            </Link>
          </InfoRow>
        )}

        {manifest?.permissions &&
          Object.keys(manifest.permissions).length > 0 && (
            <InfoRow
              label={translate('resources.plugin.fields.permissions')}
              classes={classes}
              isSmall={isSmall}
            >
              <Box className={classes.permissionsContainer}>
                {Object.entries(manifest.permissions).map(([key, value]) => (
                  <PermissionChip
                    key={key}
                    label={key}
                    permission={value}
                    classes={classes}
                  />
                ))}
              </Box>
              <Typography
                variant="caption"
                color="textSecondary"
                style={{ marginTop: 4, display: 'block' }}
              >
                {translate('resources.plugin.messages.clickPermissions')}
              </Typography>
            </InfoRow>
          )}

        <InfoRow
          label={translate('resources.plugin.fields.path')}
          classes={classes}
          isSmall={isSmall}
        >
          <span className={classes.pathField}>{record.path}</span>
        </InfoRow>

        <InfoRow
          label={translate('resources.plugin.fields.updatedAt')}
          classes={classes}
          isSmall={isSmall}
        >
          <DateField record={record} source="updatedAt" showTime />
        </InfoRow>

        <InfoRow
          label={translate('resources.plugin.fields.createdAt')}
          classes={classes}
          isSmall={isSmall}
        >
          <DateField record={record} source="createdAt" showTime />
        </InfoRow>
      </Grid>
    </CardContent>
  </Card>
)
