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

export const UsersPermissionCard = ({
  manifest,
  classes,
  selectedUsers,
  allUsers,
  onSelectedUsersChange,
  onAllUsersChange,
}) => {
  const translate = useTranslate()

  // Fetch all users
  const { data: usersData, loading: usersLoading } = useGetList('user', {
    pagination: { page: 1, perPage: 1000 },
    sort: { field: 'userName', order: 'ASC' },
  })

  const users = React.useMemo(() => {
    return usersData ? Object.values(usersData) : []
  }, [usersData])

  const handleToggleUser = React.useCallback(
    (userId) => {
      const newSelected = selectedUsers.includes(userId)
        ? selectedUsers.filter((id) => id !== userId)
        : [...selectedUsers, userId]
      onSelectedUsersChange(newSelected)
    },
    [selectedUsers, onSelectedUsersChange],
  )

  const handleAllUsersToggle = React.useCallback(
    (event) => {
      onAllUsersChange(event.target.checked)
    },
    [onAllUsersChange],
  )

  // Get permission reason from manifest
  const usersPermission = manifest?.permissions?.users
  const reason = usersPermission?.reason

  // Check if permission is required but not configured
  const isConfigurationRequired =
    usersPermission && !allUsers && selectedUsers.length === 0

  if (!usersPermission) {
    return null
  }

  return (
    <Card className={classes.section}>
      <CardContent>
        <Typography variant="h6" className={classes.sectionTitle}>
          {translate('resources.plugin.sections.usersPermission')}
        </Typography>

        {reason && (
          <Typography variant="body2" color="textSecondary" gutterBottom>
            {translate('resources.plugin.messages.permissionReason')}: {reason}
          </Typography>
        )}

        {isConfigurationRequired && (
          <Box mb={2}>
            <Alert severity="warning">
              {translate('resources.plugin.messages.usersRequired')}
            </Alert>
          </Box>
        )}

        <Box mb={2}>
          <FormControlLabel
            control={
              <Switch
                checked={allUsers}
                onChange={handleAllUsersToggle}
                color="primary"
              />
            }
            label={translate('resources.plugin.fields.allUsers')}
          />
          <Typography variant="body2" color="textSecondary">
            {translate('resources.plugin.messages.allUsersHelp')}
          </Typography>
        </Box>

        {!allUsers && (
          <Box className={classes.usersList}>
            <Typography variant="subtitle2" gutterBottom>
              {translate('resources.plugin.fields.selectedUsers')}
            </Typography>
            {usersLoading ? (
              <Typography variant="body2" color="textSecondary">
                {translate('ra.message.loading')}
              </Typography>
            ) : users.length === 0 ? (
              <Typography variant="body2" color="textSecondary">
                {translate('resources.plugin.messages.noUsers')}
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
                {users.map((user) => (
                  <ListItem
                    key={user.id}
                    button
                    onClick={() => handleToggleUser(user.id)}
                    dense
                  >
                    <ListItemIcon>
                      <Checkbox
                        icon={<CheckBoxOutlineBlankIcon fontSize="small" />}
                        checkedIcon={<CheckBoxIcon fontSize="small" />}
                        checked={selectedUsers.includes(user.id)}
                        tabIndex={-1}
                        disableRipple
                      />
                    </ListItemIcon>
                    <ListItemText
                      primary={user.name || user.userName}
                      secondary={user.name ? user.userName : null}
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

UsersPermissionCard.propTypes = {
  manifest: PropTypes.object,
  classes: PropTypes.object.isRequired,
  selectedUsers: PropTypes.array.isRequired,
  allUsers: PropTypes.bool.isRequired,
  onSelectedUsersChange: PropTypes.func.isRequired,
  onAllUsersChange: PropTypes.func.isRequired,
}
