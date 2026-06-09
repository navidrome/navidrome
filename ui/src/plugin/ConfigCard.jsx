import React, { useCallback, useState, useMemo } from 'react'
import PropTypes from 'prop-types'
import { Card, CardContent, Typography, Box } from '@material-ui/core'
import Alert from '@material-ui/lab/Alert'
import { SchemaConfigEditor } from './SchemaConfigEditor'

// Format error with field title and full path for nested fields
const formatError = (error, schema) => {
  // Get path parts from various error formats
  const rawPath =
    error.dataPath || error.property || error.instancePath?.replace(/\//g, '.')
  const parts = rawPath?.split('.').filter(Boolean) || []

  // Navigate schema to find field title, build bracket-notation path
  let currentSchema = schema
  let fieldName = parts[parts.length - 1]
  const pathParts = []

  for (const part of parts) {
    if (/^\d+$/.test(part)) {
      pathParts.push(`[${part}]`)
      currentSchema = currentSchema?.items
    } else {
      fieldName = currentSchema?.properties?.[part]?.title || part
      pathParts.push(part)
      currentSchema = currentSchema?.properties?.[part]
    }
  }

  const path = pathParts.join('.').replace(/\.\[/g, '[')
  const isNested = path.includes('[') || path.includes('.')
  // Replace property name in message with full path for nested fields
  const message = isNested
    ? error.message.replace(/'[^']+'\s*$/, `'${path}'`)
    : error.message

  return { fieldName, message }
}

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

  // Format validation errors with proper field names
  const formattedErrors = useMemo(() => {
    if (!hasConfigSchema) return []
    return validationErrors.map((error) =>
      formatError(error, manifest.config.schema),
    )
  }, [validationErrors, manifest, hasConfigSchema])

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

        {formattedErrors.length > 0 && (
          <Box mb={2}>
            <Alert severity="error">
              {translate('resources.plugin.messages.configValidationError')}
              <ul style={{ margin: '8px 0 0', paddingLeft: 20 }}>
                {formattedErrors.map((error, index) => (
                  <li key={index}>
                    {error.fieldName && <strong>{error.fieldName}</strong>}
                    {error.fieldName && ': '}
                    {error.message}
                  </li>
                ))}
              </ul>
            </Alert>
          </Box>
        )}

        <Box mt={formattedErrors.length > 0 ? 0 : 2}>
          <SchemaConfigEditor
            schema={schema}
            uiSchema={uiSchema}
            data={configData}
            onChange={handleChange}
          />
        </Box>
      </CardContent>
    </Card>
  )
}

ConfigCard.propTypes = {
  manifest: PropTypes.shape({
    config: PropTypes.shape({
      schema: PropTypes.object,
      uiSchema: PropTypes.object,
    }),
  }),
  configData: PropTypes.object,
  onConfigDataChange: PropTypes.func.isRequired,
  classes: PropTypes.object.isRequired,
  translate: PropTypes.func.isRequired,
}
