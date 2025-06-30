import { useInput, useTranslate, useRecordContext } from 'react-admin'
import { Box, FormControl, FormLabel, Typography } from '@material-ui/core'
import { SelectLibraryInput } from '../common/SelectLibraryInput.jsx'
import React, { useMemo } from 'react'

export const LibrarySelectionField = () => {
  const translate = useTranslate()
  const record = useRecordContext()

  const {
    input: { name, onChange, value },
    meta: { error, touched },
  } = useInput({ source: 'libraryIds' })

  // Extract library IDs from either 'libraries' array or 'libraryIds' array
  const libraryIds = useMemo(() => {
    // First check if form has libraryIds (create mode or already transformed)
    if (value && Array.isArray(value)) {
      return value
    }

    // Then check if record has libraries array (edit mode from backend)
    if (record?.libraries && Array.isArray(record.libraries)) {
      return record.libraries.map((lib) => lib.id)
    }

    return []
  }, [value, record])

  // Determine if this is a new user (no ID means new record)
  const isNewUser = !record?.id

  return (
    <FormControl error={!!(touched && error)} fullWidth margin="normal">
      <FormLabel component="legend">
        {translate('resources.user.fields.libraries')}
      </FormLabel>
      <Box mt={1} mb={1}>
        <SelectLibraryInput
          onChange={onChange}
          value={libraryIds}
          isNewUser={isNewUser}
        />
      </Box>
      {touched && error && (
        <Typography color="error" variant="caption">
          {error}
        </Typography>
      )}
      <Typography variant="caption" color="textSecondary">
        {translate('resources.user.helperTexts.libraries')}
      </Typography>
    </FormControl>
  )
}
