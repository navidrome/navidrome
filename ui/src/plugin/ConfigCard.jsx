import React, { useCallback, useState } from 'react'
import { Card, CardContent, Typography, Box } from '@material-ui/core'
import Alert from '@material-ui/lab/Alert'
import { SchemaConfigEditor } from './SchemaConfigEditor'

export const ConfigCard = ({
  manifest,
  configData,
  onConfigDataChange,
  classes,
  translate,
}) => {
  const [validationErrors, setValidationErrors] = useState([])

  // Handle changes from JSONForms
  const handleChange = useCallback(
    (newData, errors) => {
      setValidationErrors(errors || [])
      onConfigDataChange(newData, errors)
    },
    [onConfigDataChange],
  )

  // Only show config card if manifest has config schema defined
  const hasConfigSchema = manifest?.config?.schema
  if (!hasConfigSchema) {
    return null
  }

  const { schema, uiSchema } = manifest.config

  return (
    <Card className={classes.section}>
      <CardContent>
        <Typography variant="h6" className={classes.sectionTitle}>
          {translate('resources.plugin.sections.configuration')}
        </Typography>

        {validationErrors.length > 0 && (
          <Box mb={2}>
            <Alert severity="error">
              {translate('resources.plugin.messages.configValidationError')}
              <ul style={{ margin: '8px 0 0', paddingLeft: 20 }}>
                {validationErrors.map((error, index) => (
                  <li key={index}>
                    <strong>{error.instancePath || 'root'}</strong>:{' '}
                    {error.message}
                  </li>
                ))}
              </ul>
            </Alert>
          </Box>
        )}

        <SchemaConfigEditor
          schema={schema}
          uiSchema={uiSchema}
          data={configData}
          onChange={handleChange}
        />
      </CardContent>
    </Card>
  )
}
