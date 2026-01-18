import React from 'react'
import {
  Card,
  CardContent,
  Typography,
  Box,
  FormControlLabel,
  Switch,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Checkbox,
} from '@material-ui/core'
import CheckBoxIcon from '@material-ui/icons/CheckBox'
import CheckBoxOutlineBlankIcon from '@material-ui/icons/CheckBoxOutlineBlank'
import Alert from '@material-ui/lab/Alert'
import { useGetList, useTranslate } from 'react-admin'
import PropTypes from 'prop-types'

export const LibraryPermissionCard = ({
  manifest,
  classes,
  selectedLibraries,
  allLibraries,
  onSelectedLibrariesChange,
  onAllLibrariesChange,
}) => {
  const translate = useTranslate()

  // Fetch all libraries
  const { data: librariesData, loading: librariesLoading } = useGetList(
    'library',
    {
      pagination: { page: 1, perPage: 1000 },
      sort: { field: 'name', order: 'ASC' },
    },
  )

  const libraries = React.useMemo(() => {
    return librariesData ? Object.values(librariesData) : []
  }, [librariesData])

  const handleToggleLibrary = React.useCallback(
    (libraryId) => {
      const newSelected = selectedLibraries.includes(libraryId)
        ? selectedLibraries.filter((id) => id !== libraryId)
        : [...selectedLibraries, libraryId]
      onSelectedLibrariesChange(newSelected)
    },
    [selectedLibraries, onSelectedLibrariesChange],
  )

  const handleAllLibrariesToggle = React.useCallback(
    (event) => {
      onAllLibrariesChange(event.target.checked)
    },
    [onAllLibrariesChange],
  )

  // Get permission reason from manifest
  const libraryPermission = manifest?.permissions?.library
  const reason = libraryPermission?.reason

  // Check if permission is required but not configured
  const isConfigurationRequired =
    libraryPermission && !allLibraries && selectedLibraries.length === 0

  if (!libraryPermission) {
    return null
  }

  return (
    <Card className={classes.section}>
      <CardContent>
        <Typography variant="h6" className={classes.sectionTitle}>
          {translate('resources.plugin.sections.libraryPermission')}
        </Typography>

        {reason && (
          <Typography variant="body2" color="textSecondary" gutterBottom>
            {translate('resources.plugin.messages.permissionReason')}: {reason}
          </Typography>
        )}

        {isConfigurationRequired && (
          <Box mb={2}>
            <Alert severity="warning">
              {translate('resources.plugin.messages.librariesRequired')}
            </Alert>
          </Box>
        )}

        <Box mb={2}>
          <FormControlLabel
            control={
              <Switch
                checked={allLibraries}
                onChange={handleAllLibrariesToggle}
                color="primary"
              />
            }
            label={translate('resources.plugin.fields.allLibraries')}
          />
          <Typography variant="body2" color="textSecondary">
            {translate('resources.plugin.messages.allLibrariesHelp')}
          </Typography>
        </Box>

        {!allLibraries && (
          <Box className={classes.usersList}>
            <Typography variant="subtitle2" gutterBottom>
              {translate('resources.plugin.fields.selectedLibraries')}
            </Typography>
            {librariesLoading ? (
              <Typography variant="body2" color="textSecondary">
                {translate('ra.message.loading')}
              </Typography>
            ) : libraries.length === 0 ? (
              <Typography variant="body2" color="textSecondary">
                {translate('resources.plugin.messages.noLibraries')}
              </Typography>
            ) : (
              <List
                dense
                style={{
                  maxHeight: 200,
                  overflow: 'auto',
                  border: '1px solid rgba(0, 0, 0, 0.12)',
                  borderRadius: 4,
                }}
              >
                {libraries.map((library) => (
                  <ListItem
                    key={library.id}
                    button
                    onClick={() => handleToggleLibrary(library.id)}
                    dense
                  >
                    <ListItemIcon>
                      <Checkbox
                        icon={<CheckBoxOutlineBlankIcon fontSize="small" />}
                        checkedIcon={<CheckBoxIcon fontSize="small" />}
                        checked={selectedLibraries.includes(library.id)}
                        tabIndex={-1}
                        disableRipple
                      />
                    </ListItemIcon>
                    <ListItemText
                      primary={library.name}
                      secondary={library.path}
                    />
                  </ListItem>
                ))}
              </List>
            )}
          </Box>
        )}
      </CardContent>
    </Card>
  )
}

LibraryPermissionCard.propTypes = {
  manifest: PropTypes.object,
  classes: PropTypes.object.isRequired,
  selectedLibraries: PropTypes.array.isRequired,
  allLibraries: PropTypes.bool.isRequired,
  onSelectedLibrariesChange: PropTypes.func.isRequired,
  onAllLibrariesChange: PropTypes.func.isRequired,
}
