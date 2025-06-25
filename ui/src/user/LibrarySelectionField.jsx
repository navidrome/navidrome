import { useInput, useTranslate } from 'react-admin'
import { Box, FormControl, FormLabel, Typography } from '@material-ui/core'
import { SelectLibraryInput } from '../common/SelectLibraryInput.jsx'
import React from 'react'

export const LibrarySelectionField = () => {
  const translate = useTranslate()
  const {
    input: { name, onChange, value },
    meta: { error, touched },
  } = useInput({ source: 'libraryIds' })

  return (
    <FormControl error={!!(touched && error)} fullWidth margin="normal">
      <FormLabel component="legend" required>
        {translate('resources.user.fields.libraries')}
      </FormLabel>
      <Box mt={1} mb={1}>
        <SelectLibraryInput onChange={onChange} value={value || []} />
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
