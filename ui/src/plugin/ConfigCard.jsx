import React, { useCallback, useState, useMemo } from 'react'
import PropTypes from 'prop-types'
import { Card, CardContent, Typography, Box } from '@material-ui/core'
import Alert from '@material-ui/lab/Alert'
import { SchemaConfigEditor } from './SchemaConfigEditor'

// Navigate schema by path parts to find the title for a field
const findFieldTitle = (schema, parts) => {
  let currentSchema = schema
  let fieldName = parts[parts.length - 1] // Default to last part

  for (const part of parts) {
    if (!currentSchema) break

    // Skip array indices (just move to items schema)
    if (/^\d+$/.test(part)) {
      if (currentSchema.items) {
        currentSchema = currentSchema.items
      }
      continue
    }

    // Navigate to property and always update fieldName
    if (currentSchema.properties?.[part]) {
      const propSchema = currentSchema.properties[part]
      fieldName = propSchema.title || part
      currentSchema = propSchema
    }
  }

  return fieldName
}

// Extract human-readable field name from JSONForms error
const getFieldName = (error, schema) => {
  // JSONForms errors can have different path formats:
  // - dataPath: "users.1.token" (dot-separated)
  // - instancePath: "/users/1/token" (slash-separated)
  // - property: "users.1.username" (dot-separated)
  const dataPath = error.dataPath || ''
  const instancePath = error.instancePath || ''
  const property = error.property || ''

  // Try dataPath first (dot-separated like "users.1.token")
  if (dataPath) {
    const parts = dataPath.split('.').filter(Boolean)
    if (parts.length > 0) {
      return findFieldTitle(schema, parts)
    }
  }

  // Try property (also dot-separated)
  if (property) {
    const parts = property.split('.').filter(Boolean)
    if (parts.length > 0) {
      return findFieldTitle(schema, parts)
    }
  }

  // Fall back to instancePath (slash-separated like "/users/1/token")
  if (instancePath) {
    const parts = instancePath.split('/').filter(Boolean)
    if (parts.length > 0) {
      return findFieldTitle(schema, parts)
    }
  }

  // Try to extract from schemaPath like "#/properties/users/items/properties/username/minLength"
  const schemaPath = error.schemaPath || ''
  const propMatches = [...schemaPath.matchAll(/\/properties\/([^/]+)/g)]
  if (propMatches.length > 0) {
    const parts = propMatches.map((m) => m[1])
    return findFieldTitle(schema, parts)
  }

  return null
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
    if (!hasConfigSchema) {
      return []
    }
    const { schema } = manifest.config
    return validationErrors.map((error) => ({
      fieldName: getFieldName(error, schema),
      message: error.message,
    }))
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
