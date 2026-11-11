import React from 'react'
import PropTypes from 'prop-types'
import get from 'lodash/get'
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
    inputRoot: {
      '&:hover $notchedOutline': {
        borderColor: theme.palette.divider,
      },
    },
    notchedOutline: {
      borderColor: theme.palette.divider,
    },
  }),
  { name: 'NDReadOnlyField' },
)

const identity = (v) => v

// Renders a record value as a dimmed, non-editable input, so it lines up with the inputs in a form
export const ReadOnlyTextField = ({
  source,
  label,
  resource,
  className,
  fullWidth,
  format = identity,
  ...props
}) => {
  const classes = useStyles(props)
  const record = useRecordContext(props)
  const value = get(record, source)

  return (
    <TextField
      id={source}
      className={className}
      label={<FieldTitle label={label} source={source} resource={resource} />}
      value={value == null ? '' : format(value)}
      variant="outlined"
      margin="dense"
      fullWidth={fullWidth}
      focused={false}
      helperText=" "
      InputProps={{
        readOnly: true,
        classes: {
          root: classes.inputRoot,
          notchedOutline: classes.notchedOutline,
        },
      }}
      inputProps={{ tabIndex: -1 }}
    />
  )
}

ReadOnlyTextField.propTypes = {
  source: PropTypes.string.isRequired,
  label: PropTypes.oneOfType([PropTypes.string, PropTypes.bool]),
  record: PropTypes.object,
  resource: PropTypes.string,
  className: PropTypes.string,
  classes: PropTypes.object,
  fullWidth: PropTypes.bool,
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
