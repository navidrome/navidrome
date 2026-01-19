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
import merge from 'lodash/merge'

/**
 * Hook for common control state (focus, validation, description visibility)
 */
const useControlState = (props) => {
  const { config, uischema, errors, description, visible } = props
  const [isFocused, setIsFocused] = useState(false)

  const appliedUiSchemaOptions = merge({}, config, uischema?.options)
  const isValid = errors?.length === 0

  const showDescription = !isDescriptionHidden(
    visible,
    description,
    isFocused,
    appliedUiSchemaOptions.showUnfocusedDescription,
  )

  const helperText = !isValid ? errors : showDescription ? description : ''

  return {
    isFocused,
    setIsFocused,
    appliedUiSchemaOptions,
    isValid,
    helperText,
  }
}

/**
 * Base outlined control component that uses TextField with outlined variant
 * instead of the default Input component used by JSONForms 2.x
 */
const OutlinedControl = (props) => {
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

  const { setIsFocused, appliedUiSchemaOptions, isValid, helperText } =
    useControlState(props)

  if (!visible) {
    return null
  }

  return (
    <FormControl fullWidth error={!isValid}>
      <TextField
        id={id}
        label={label}
        type={type}
        value={data ?? ''}
        onChange={onChange}
        onFocus={() => setIsFocused(true)}
        onBlur={() => setIsFocused(false)}
        disabled={!enabled}
        autoFocus={appliedUiSchemaOptions.focus}
        multiline={type === 'text' && appliedUiSchemaOptions.multi}
        rows={appliedUiSchemaOptions.multi ? 3 : undefined}
        variant="outlined"
        fullWidth
        size="small"
        error={!isValid}
        helperText={helperText}
        inputProps={extraInputProps}
      />
    </FormControl>
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
  const { data, id, enabled, path, handleChange, options, label, visible } =
    props
  const { setIsFocused, appliedUiSchemaOptions, isValid, helperText } =
    useControlState(props)

  if (!visible) {
    return null
  }

  return (
    <FormControl fullWidth variant="outlined" size="small" error={!isValid}>
      <InputLabel id={`${id}-label`}>{label}</InputLabel>
      <Select
        labelId={`${id}-label`}
        id={id}
        value={data ?? ''}
        onChange={(ev) => handleChange(path, ev.target.value)}
        onFocus={() => setIsFocused(true)}
        onBlur={() => setIsFocused(false)}
        disabled={!enabled}
        autoFocus={appliedUiSchemaOptions.focus}
        label={label}
        fullWidth
      >
        <MenuItem value="">
          <em>None</em>
        </MenuItem>
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
