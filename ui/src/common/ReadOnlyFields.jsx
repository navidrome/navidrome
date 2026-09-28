import React from 'react'
import PropTypes from 'prop-types'
import get from 'lodash/get'
import clsx from 'clsx'
import { FieldTitle, useRecordContext } from 'react-admin'
import { TextField } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { useDateLocale } from '../i18n/useDateLocale'
import {
  formatBytes,
  formatDateTime,
  formatDuration2,
  formatNumber,
} from '../utils/formatters'
import { isDateSet } from '../utils/validations'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      '& .MuiOutlinedInput-root .MuiOutlinedInput-notchedOutline': {
        borderColor: theme.palette.divider,
        borderWidth: 1,
      },
      '& .MuiInputLabel-root.Mui-focused': {
        color: theme.palette.text.secondary,
      },
    },
  }),
  { name: 'NDReadOnlyField' },
)

const identity = (v) => v

// Renders a record value as a dimmed, non-editable input, so it lines up with the inputs in a form
export const ReadOnlyTextField = ({
  id,
  source,
  label,
  resource,
  className,
  fullWidth,
  margin = 'dense',
  format = identity,
  ...props
}) => {
  const classes = useStyles(props)
  const record = useRecordContext(props)
  const value = get(record, source)

  return (
    <TextField
      id={id || source}
      className={clsx(classes.root, className)}
      label={<FieldTitle label={label} source={source} resource={resource} />}
      value={value == null ? '' : format(value)}
      variant="outlined"
      margin={margin}
      fullWidth={fullWidth}
      helperText=" "
      InputProps={{ readOnly: true }}
      inputProps={{ tabIndex: -1 }}
    />
  )
}

ReadOnlyTextField.propTypes = {
  id: PropTypes.string,
  source: PropTypes.string.isRequired,
  label: PropTypes.oneOfType([PropTypes.string, PropTypes.bool]),
  record: PropTypes.object,
  resource: PropTypes.string,
  className: PropTypes.string,
  classes: PropTypes.object,
  fullWidth: PropTypes.bool,
  margin: PropTypes.oneOf(['none', 'dense', 'normal']),
  format: PropTypes.func,
}

export const ReadOnlyDateField = (props) => {
  const locale = useDateLocale()
  const format = (v) => (isDateSet(v) ? formatDateTime(v, locale) : '')
  return <ReadOnlyTextField format={format} {...props} />
}

export const ReadOnlyNumberField = (props) => {
  const locale = useDateLocale()
  return (
    <ReadOnlyTextField format={(v) => formatNumber(v, locale)} {...props} />
  )
}

export const ReadOnlySizeField = (props) => (
  <ReadOnlyTextField format={formatBytes} {...props} />
)

export const ReadOnlyDurationField = (props) => (
  <ReadOnlyTextField format={formatDuration2} {...props} />
)
