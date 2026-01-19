import React, { useCallback, useEffect, useMemo, useRef } from 'react'
import PropTypes from 'prop-types'
import { JsonForms } from '@jsonforms/react'
import { materialRenderers, materialCells } from '@jsonforms/material-renderers'
import { makeStyles } from '@material-ui/core/styles'
import { Typography } from '@material-ui/core'
import { useTranslate } from 'react-admin'
import Ajv from 'ajv'
import { AlwaysExpandedArrayLayout } from './AlwaysExpandedArrayLayout'

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

// Create AJV instance with useDefaults to auto-apply schema defaults
const ajv = new Ajv({
  useDefaults: true,
  allErrors: true,
  verbose: true,
})

const useStyles = makeStyles(
  (theme) => ({
    root: {
      // Override JSONForms to use outlined-style inputs (matching Navidrome's design)
      '& .MuiFormControl-root': {
        marginBottom: theme.spacing(2),
      },
      // Style the Input to look like outlined variant
      '& .MuiInput-root': {
        position: 'relative',
        border: `1px solid ${theme.palette.type === 'dark' ? 'rgba(255, 255, 255, 0.23)' : 'rgba(0, 0, 0, 0.23)'}`,
        borderRadius: theme.shape.borderRadius,
        padding: '10px 14px',
        marginTop: theme.spacing(2),
        '&:hover': {
          borderColor: theme.palette.text.primary,
        },
        '&.Mui-focused': {
          borderColor: theme.palette.primary.main,
          borderWidth: 2,
          padding: '9px 13px', // Adjust for border width change
        },
        '&.Mui-error': {
          borderColor: theme.palette.error.main,
        },
        '&:before, &:after': {
          display: 'none', // Hide the default underline
        },
      },
      '& .MuiInput-input': {
        padding: 0,
      },
      // Position label like outlined variant (floating above the border)
      '& .MuiInputLabel-root': {
        position: 'absolute',
        left: 0,
        top: 0,
        transform: 'translate(14px, 28px) scale(1)', // Accounts for marginTop on input
        transformOrigin: 'top left',
        transition: theme.transitions.create(['transform', 'color'], {
          duration: theme.transitions.duration.shorter,
        }),
        '&.MuiInputLabel-shrink': {
          transform: 'translate(14px, 7px) scale(0.75)',
          backgroundColor: theme.palette.background.paper,
          padding: '0 5px',
          marginLeft: '-5px',
          zIndex: 1,
          // Use box-shadow to extend background coverage and hide border
          boxShadow: `0 0 0 3px ${theme.palette.background.paper}`,
        },
        '&.Mui-focused': {
          color: theme.palette.primary.main,
        },
        '&.Mui-error': {
          color: theme.palette.error.main,
        },
      },
      '& .MuiFormHelperText-root': {
        marginTop: theme.spacing(0.5),
        marginLeft: theme.spacing(1.75),
      },
      '& .MuiFormHelperText-root.Mui-error': {
        color: theme.palette.error.main,
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

// Custom renderers with always-expanded array layout
const customRenderers = [
  // Put our custom renderer first (higher priority)
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

  // JSONForms config - always show descriptions to avoid layout shifts
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
