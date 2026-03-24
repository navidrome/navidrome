/* eslint-disable react-refresh/only-export-components */
import React, { useState } from 'react'
import {
  rankWith,
  isStringControl,
  isIntegerControl,
  isNumberControl,
  isEnumControl,
  isOneOfEnumControl,
  and,
  not,
  or,
  optionIs,
  isDescriptionHidden,
} from '@jsonforms/core'
import {
  withJsonFormsControlProps,
  withJsonFormsEnumProps,
  withJsonFormsOneOfEnumProps,
} from '@jsonforms/react'
import {
  TextField,
  FormControl,
  FormHelperText,
  InputLabel,
  Select,
  MenuItem,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import merge from 'lodash/merge'

const useStyles = makeStyles(
  (theme) => ({
    control: {
      marginBottom: theme.spacing(2),
    },
  }),
  { name: 'NDOutlinedRenderers' },
)

/**
 * Hook for common control state (focus, validation, description visibility)
 */
const useControlState = (props) => {
  const { config, uischema, description, visible, errors } = props
  const [isFocused, setIsFocused] = useState(false)

  const appliedUiSchemaOptions = merge({}, config, uischema?.options)
  // errors is a string when there are validation errors, empty/undefined when valid
  const showError = errors && errors.length > 0

  const showDescription = !isDescriptionHidden(
    visible,
    description,
    isFocused,
    appliedUiSchemaOptions.showUnfocusedDescription,
  )

  const helperText = showError ? errors : showDescription ? description : ''

  const handleFocus = () => setIsFocused(true)
  const handleBlur = () => setIsFocused(false)

  return {
    isFocused,
    appliedUiSchemaOptions,
    showError,
    helperText,
    handleFocus,
    handleBlur,
  }
}

/**
 * Base outlined control component that uses TextField with outlined variant
 * instead of the default Input component used by JSONForms 2.x
 */
const OutlinedControl = (props) => {
  const classes = useStyles()
  const {
    data,
    id,
    enabled,
    label,
    visible,
    type = 'text',
    inputProps: extraInputProps = {},
    onChange,
  } = props

  const {
    appliedUiSchemaOptions,
    showError,
    helperText,
    handleFocus,
    handleBlur,
  } = useControlState(props)

  if (!visible) {
    return null
  }

  return (
    <TextField
      id={id}
      label={label}
      type={type}
      value={data ?? ''}
      onChange={onChange}
      onFocus={handleFocus}
      onBlur={handleBlur}
      disabled={!enabled}
      autoFocus={appliedUiSchemaOptions.focus}
      multiline={type === 'text' && appliedUiSchemaOptions.multi}
      rows={appliedUiSchemaOptions.multi ? 3 : undefined}
      variant="outlined"
      fullWidth
      size="small"
      error={showError}
      helperText={helperText}
      inputProps={extraInputProps}
      className={classes.control}
    />
  )
}

// Text control wrapper
const OutlinedTextControl = (props) => {
  const { path, handleChange, schema, config, uischema } = props
  const appliedUiSchemaOptions = merge({}, config, uischema?.options)

  const inputProps = {}
  if (appliedUiSchemaOptions.restrict && schema?.maxLength) {
    inputProps.maxLength = schema.maxLength
  }

  return (
    <OutlinedControl
      {...props}
      type={appliedUiSchemaOptions.format === 'password' ? 'password' : 'text'}
      inputProps={inputProps}
      onChange={(ev) => handleChange(path, ev.target.value)}
    />
  )
}

// Number control wrapper
const OutlinedNumberControl = (props) => {
  const { path, handleChange, schema } = props
  const { minimum, maximum } = schema || {}

  const inputProps = {}
  if (minimum !== undefined) inputProps.min = minimum
  if (maximum !== undefined) inputProps.max = maximum

  const handleNumberChange = (ev) => {
    const value = ev.target.value
    if (value === '') {
      handleChange(path, undefined)
    } else {
      const numValue = Number(value)
      if (!isNaN(numValue)) {
        handleChange(path, numValue)
      }
    }
  }

  return (
    <OutlinedControl
      {...props}
      type="number"
      inputProps={inputProps}
      onChange={handleNumberChange}
    />
  )
}

// Enum/Select control wrapper
const OutlinedEnumControl = (props) => {
  const classes = useStyles()
  const {
    data,
    id,
    enabled,
    path,
    handleChange,
    options,
    label,
    visible,
    required,
  } = props
  const {
    appliedUiSchemaOptions,
    showError,
    helperText,
    handleFocus,
    handleBlur,
  } = useControlState(props)

  if (!visible) {
    return null
  }

  return (
    <FormControl
      fullWidth
      variant="outlined"
      size="small"
      error={showError}
      className={classes.control}
    >
      <InputLabel id={`${id}-label`}>{label}</InputLabel>
      <Select
        labelId={`${id}-label`}
        id={id}
        value={data ?? ''}
        onChange={(ev) => {
          handleChange(
            path,
            ev.target.value === '' ? undefined : ev.target.value,
          )
        }}
        onFocus={handleFocus}
        onBlur={handleBlur}
        disabled={!enabled}
        autoFocus={appliedUiSchemaOptions.focus}
        label={label}
        fullWidth
      >
        {!required && (
          <MenuItem value="">
            <em>None</em>
          </MenuItem>
        )}
        {options?.map((option) => (
          <MenuItem key={option.value} value={option.value}>
            {option.label}
          </MenuItem>
        ))}
      </Select>
      {helperText && <FormHelperText>{helperText}</FormHelperText>}
    </FormControl>
  )
}

// Testers - higher rank than default to override default renderers
// Enum renderers have highest rank since isStringControl also matches enum fields
export const OutlinedEnumRenderer = {
  tester: rankWith(5, isEnumControl),
  renderer: withJsonFormsEnumProps(OutlinedEnumControl),
}

export const OutlinedOneOfEnumRenderer = {
  tester: rankWith(5, isOneOfEnumControl),
  renderer: withJsonFormsOneOfEnumProps(OutlinedEnumControl),
}

export const OutlinedTextRenderer = {
  tester: rankWith(3, and(isStringControl, not(optionIs('format', 'radio')))),
  renderer: withJsonFormsControlProps(OutlinedTextControl),
}

export const OutlinedNumberRenderer = {
  tester: rankWith(3, or(isIntegerControl, isNumberControl)),
  renderer: withJsonFormsControlProps(OutlinedNumberControl),
}
