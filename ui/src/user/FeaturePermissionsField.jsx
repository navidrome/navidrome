import { useInput, useTranslate, useRecordContext } from 'react-admin'
import {
  Box,
  FormControl,
  FormControlLabel,
  FormLabel,
  Checkbox,
  Typography,
} from '@material-ui/core'
import React, { useMemo } from 'react'
import { ALL_USER_FEATURES } from '../consts'

export const FeaturePermissionsField = () => {
  const translate = useTranslate()
  const record = useRecordContext()

  const {
    input: { value, onChange },
  } = useInput({ source: 'featurePermissions' })

  // Same shape react-admin gives libraryIds: prefer the form's current (possibly-edited) value,
  // fall back to the record as loaded. A feature absent from either defaults to enabled, matching
  // the backend's opt-out default (model.AllUserFeatures in model/user_feature_permission.go).
  const permissions = useMemo(() => {
    const source = value || record?.featurePermissions || {}
    return ALL_USER_FEATURES.reduce((acc, feature) => {
      acc[feature] = source[feature] !== false
      return acc
    }, {})
  }, [value, record])

  const toggle = (feature) => (event) => {
    onChange({ ...permissions, [feature]: event.target.checked })
  }

  return (
    <FormControl fullWidth margin="normal">
      <FormLabel component="legend">
        {translate('resources.user.fields.featurePermissions')}
      </FormLabel>
      <Box mt={1} mb={1}>
        {ALL_USER_FEATURES.map((feature) => (
          <FormControlLabel
            key={feature}
            control={
              <Checkbox
                id={`featurePermissions-${feature}`}
                color="primary"
                checked={permissions[feature]}
                onChange={toggle(feature)}
              />
            }
            label={translate(`resources.user.featurePermissions.${feature}`)}
          />
        ))}
      </Box>
      <Typography variant="caption" color="textSecondary">
        {translate('resources.user.helperTexts.featurePermissions')}
      </Typography>
    </FormControl>
  )
}
