import React, { useCallback, useEffect, useMemo, useRef } from 'react'
import PropTypes from 'prop-types'
import { JsonForms } from '@jsonforms/react'
import { materialRenderers, materialCells } from '@jsonforms/material-renderers'
import { makeStyles } from '@material-ui/core/styles'
import { Typography } from '@material-ui/core'
import { useTranslate } from 'react-admin'
import Ajv from 'ajv'
import { AlwaysExpandedArrayLayout } from './AlwaysExpandedArrayLayout'
import {
  OutlinedTextRenderer,
  OutlinedNumberRenderer,
  OutlinedEnumRenderer,
  OutlinedOneOfEnumRenderer,
} from './OutlinedRenderers'

// Error boundary for catching JSONForms rendering errors
class SchemaErrorBoundary extends React.Component {
  constructor(props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback(this.state.error)
    }
    return this.props.children
  }
}

SchemaErrorBoundary.propTypes = {
  children: PropTypes.node.isRequired,
  fallback: PropTypes.func.isRequired,
}

// Custom AJV instance that fixes "required" error paths for JSONForms.
// AJV outputs required errors pointing to the parent (e.g., "/users/1") with
// params.missingProperty. We transform them to point to the field directly
// (e.g., "/users/1/username") so JSONForms displays them under the correct input.
const ajv = new Ajv({
  useDefaults: true,
  allErrors: true,
  verbose: true,
  jsonPointers: true,
})
const origCompile = ajv.compile.bind(ajv)
ajv.compile = (schema) => {
  const validate = origCompile(schema)
  const wrapped = (data) => {
    const valid = validate(data)
    validate.errors?.forEach((e) => {
      if (e.keyword === 'required' && e.params?.missingProperty) {
        e.dataPath = `${e.dataPath || ''}/${e.params.missingProperty}`
      }
    })
    wrapped.errors = validate.errors
    return valid
  }
  wrapped.schema = validate.schema
  return wrapped
}

const useStyles = makeStyles(
  (theme) => ({
    root: {
      '& .MuiFormControl-root': {
        marginBottom: theme.spacing(2),
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
        marginBottom: theme.spacing(1),
        '&:before': {
          display: 'none',
        },
      },
      '& .MuiAccordionSummary-root': {
        backgroundColor:
          theme.palette.type === 'dark'
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
    },
    errorContainer: {
      padding: theme.spacing(2),
      backgroundColor:
        theme.palette.type === 'dark'
          ? 'rgba(244, 67, 54, 0.1)'
          : 'rgba(244, 67, 54, 0.05)',
      borderRadius: theme.shape.borderRadius,
      border: `1px solid ${theme.palette.error.main}`,
    },
    errorMessage: {
      color: theme.palette.error.main,
      marginBottom: theme.spacing(1),
    },
    errorDetails: {
      color: theme.palette.text.secondary,
      fontSize: '0.85em',
      fontFamily: 'monospace',
      whiteSpace: 'pre-wrap',
      wordBreak: 'break-word',
    },
  }),
  { name: 'NDSchemaConfigEditor' },
)

// Custom renderers with outlined text inputs and always-expanded array layout
const customRenderers = [
  // Put our custom renderers first (higher priority)
  OutlinedTextRenderer,
  OutlinedNumberRenderer,
  OutlinedEnumRenderer,
  OutlinedOneOfEnumRenderer,
  AlwaysExpandedArrayLayout,
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
  const classes = useStyles()
  const translate = useTranslate()
  const containerRef = useRef(null)

  // Disable browser autocomplete on all inputs
  useEffect(() => {
    if (!containerRef.current) return

    const disableAutocomplete = () => {
      const inputs = containerRef.current.querySelectorAll('input')
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
    <div className={classes.errorContainer}>
      <Typography className={classes.errorMessage}>
        {translate('resources.plugin.messages.schemaRenderError')}
      </Typography>
      <Typography className={classes.errorDetails}>{error?.message}</Typography>
    </div>
  )

  return (
    <div ref={containerRef} className={classes.root}>
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
    </div>
  )
}

SchemaConfigEditor.propTypes = {
  schema: PropTypes.object,
  uiSchema: PropTypes.object,
  data: PropTypes.object,
  onChange: PropTypes.func,
  readOnly: PropTypes.bool,
}
