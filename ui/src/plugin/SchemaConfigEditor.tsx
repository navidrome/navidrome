import React, { useCallback, useEffect, useMemo, useRef } from 'react'
import { JsonForms } from '@jsonforms/react'
import { materialRenderers, materialCells } from '@jsonforms/material-renderers'
import { Box, Typography } from '@mui/material'
import { useTranslate } from 'react-admin'
import Ajv from 'ajv'
import {
  OutlinedTextRenderer,
  OutlinedNumberRenderer,
  OutlinedEnumRenderer,
  OutlinedOneOfEnumRenderer,
} from './OutlinedRenderers'
import { componentStyleOverride } from '../themes/componentStyleOverride'

// Error boundary for catching JSONForms rendering errors
type SchemaErrorBoundaryProps = {
  children: React.ReactNode
  fallback: (error: Error) => React.ReactNode
}

type SchemaErrorBoundaryState = {
  hasError: boolean
  error: Error | null
}

class SchemaErrorBoundary extends React.Component<
  SchemaErrorBoundaryProps,
  SchemaErrorBoundaryState
> {
  constructor(props: SchemaErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback(this.state.error as Error)
    }
    return this.props.children
  }
}

// Custom AJV instance that fixes "required" error paths for JSONForms.
// AJV outputs required errors pointing to the parent (e.g., "/users/1") with
// params.missingProperty. We transform them to point to the field directly
// (e.g., "/users/1/username") so JSONForms displays them under the correct input.
const ajv = new Ajv({
  useDefaults: true,
  allErrors: true,
  verbose: true,
})
const origCompile = ajv.compile.bind(ajv)
ajv.compile = ((schema) => {
  const validate = origCompile(schema)
  const wrapped = ((data: unknown) => {
    const valid = validate(data)
    validate.errors?.forEach((e) => {
      const params = e.params as { missingProperty?: string }
      if (e.keyword === 'required' && params?.missingProperty) {
        // ajv@8 renamed dataPath -> instancePath (JSON pointer)
        e.instancePath = `${e.instancePath || ''}/${params.missingProperty}`
      }
    })
    wrapped.errors = validate.errors
    return valid
  }) as typeof validate
  wrapped.schema = validate.schema
  wrapped.schemaEnv = validate.schemaEnv
  return wrapped
}) as typeof ajv.compile

const rootSx = (theme) => ({
  '& .MuiFormControl-root': {
    mb: 2,
  },
  // Label elements (type: "Label" in UI schema) - make slightly smaller
  '& .MuiTypography-h6': {
    fontSize: '0.95rem',
  },
  // Group/array styling
  '& .MuiPaper-root': {
    backgroundColor: 'transparent',
  },
  // Array items styling
  '& .MuiAccordion-root': {
    mb: 1,
    '&:before': {
      display: 'none',
    },
  },
  '& .MuiAccordionSummary-root': {
    backgroundColor:
      theme.palette.mode === 'dark'
        ? theme.palette.grey[800]
        : theme.palette.grey[100],
    // Hide expand icon - items are always expanded
    '& .MuiAccordionSummary-expandIcon': {
      display: 'none',
    },
  },
  // Checkbox/switch styling
  '& .MuiCheckbox-root, & .MuiSwitch-root': {
    color: theme.palette.text.secondary,
  },
  '& .Mui-checked': {
    color: theme.palette.primary.main,
  },
  ...componentStyleOverride(theme, 'NDSchemaConfigEditor', 'root'),
})

// Custom renderers with outlined text inputs and always-expanded array layout
const customRenderers = [
  // Put our custom renderers first (higher priority)
  OutlinedTextRenderer,
  OutlinedNumberRenderer,
  OutlinedEnumRenderer,
  OutlinedOneOfEnumRenderer,
  // Then all the standard material renderers
  ...materialRenderers,
]

export const SchemaConfigEditor = ({
  schema,
  uiSchema,
  data,
  onChange,
  readOnly = false,
}) => {
  const translate = useTranslate()
  const containerRef = useRef<HTMLDivElement | null>(null)

  // Disable browser autocomplete on all inputs
  useEffect(() => {
    if (!containerRef.current) return

    const disableAutocomplete = () => {
      const container = containerRef.current
      if (!container) return
      const inputs = container.querySelectorAll('input')
      inputs.forEach((input) => {
        input.setAttribute('autocomplete', 'off')
      })
    }

    // Run immediately and observe for changes (new inputs added)
    disableAutocomplete()
    const observer = new MutationObserver(disableAutocomplete)
    observer.observe(containerRef.current, { childList: true, subtree: true })

    return () => observer.disconnect()
  }, [data])

  // Memoize the change handler to extract just the data
  const handleChange = useCallback(
    ({ data: newData, errors }) => {
      if (onChange) {
        onChange(newData, errors)
      }
    },
    [onChange],
  )

  // Use custom renderers with always-expanded array layout
  const renderers = useMemo(() => customRenderers, [])
  const cells = useMemo(() => materialCells, [])

  // JSONForms config - always show descriptions
  const config = {
    showUnfocusedDescription: true,
  }

  // Ensure schema has required fields for JSONForms
  const normalizedSchema = useMemo(() => {
    if (!schema) return null
    // JSONForms requires type to be set at root level
    return {
      type: 'object',
      ...schema,
    }
  }, [schema])

  if (!normalizedSchema) {
    return null
  }

  const renderError = (error) => (
    <Box
      sx={(theme) => ({
        p: 2,
        backgroundColor:
          theme.palette.mode === 'dark'
            ? 'rgba(244, 67, 54, 0.1)'
            : 'rgba(244, 67, 54, 0.05)',
        borderRadius: 1,
        border: `1px solid ${theme.palette.error.main}`,
        ...componentStyleOverride(
          theme,
          'NDSchemaConfigEditor',
          'errorContainer',
        ),
      })}
    >
      <Typography
        sx={(theme) => ({
          color: theme.palette.error.main,
          mb: 1,
          ...componentStyleOverride(
            theme,
            'NDSchemaConfigEditor',
            'errorMessage',
          ),
        })}
      >
        {translate('resources.plugin.messages.schemaRenderError')}
      </Typography>
      <Typography
        sx={(theme) => ({
          color: theme.palette.text.secondary,
          fontSize: '0.85em',
          fontFamily: 'monospace',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          ...componentStyleOverride(
            theme,
            'NDSchemaConfigEditor',
            'errorDetails',
          ),
        })}
      >
        {error?.message}
      </Typography>
    </Box>
  )

  return (
    <Box ref={containerRef} className="NDSchemaConfigEditor-root" sx={rootSx}>
      <SchemaErrorBoundary fallback={renderError}>
        <JsonForms
          schema={normalizedSchema}
          uischema={uiSchema}
          data={data || {}}
          renderers={renderers}
          cells={cells}
          config={config}
          onChange={handleChange}
          readonly={readOnly}
          ajv={ajv}
          validationMode="ValidateAndShow"
        />
      </SchemaErrorBoundary>
    </Box>
  )
}
