import React from 'react'
import PropTypes from 'prop-types'
import get from 'lodash/get'
import clsx from 'clsx'
import { FieldTitle, useRecordContext } from 'react-admin'
import { TextField } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'

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

// Renders a record value as a dimmed, non-editable input, so it lines up with the inputs in a form
export const ReadOnlyField = ({
  id,
  source,
  label,
  resource,
  className,
  fullWidth,
  margin = 'dense',
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
      value={value ?? ''}
      variant="outlined"
      margin={margin}
      fullWidth={fullWidth}
      helperText=" "
      InputProps={{ readOnly: true }}
      inputProps={{ tabIndex: -1 }}
    />
  )
}

ReadOnlyField.propTypes = {
  id: PropTypes.string,
  source: PropTypes.string.isRequired,
  label: PropTypes.oneOfType([PropTypes.string, PropTypes.bool]),
  record: PropTypes.object,
  resource: PropTypes.string,
  className: PropTypes.string,
  classes: PropTypes.object,
  fullWidth: PropTypes.bool,
  margin: PropTypes.oneOf(['none', 'dense', 'normal']),
}
