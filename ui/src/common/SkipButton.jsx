import React, { useCallback } from 'react'
import PropTypes from 'prop-types'
import BlockIcon from '@material-ui/icons/Block'
import IconButton from '@material-ui/core/IconButton'
import { makeStyles } from '@material-ui/core/styles'
import clsx from 'clsx'
import { useToggleSkip } from './useToggleSkip'
import { useRecordContext } from 'react-admin'
import { isDateSet } from '../utils/validations'

const useStyles = makeStyles(
  {
    skip: {
      color: (props) => props.color,
      opacity: (props) => (props.skipped ? 1 : 0.4),
      visibility: (props) =>
        props.visible === false
          ? 'hidden'
          : props.skipped
            ? 'visible'
            : 'inherit',
    },
  },
  { name: 'NDSkipButton' },
)

export const SkipButton = ({
  resource,
  color,
  visible,
  size,
  component: Button,
  addLabel,
  disabled,
  className,
  record: recordProp,
  ...rest
}) => {
  const record = useRecordContext({ record: recordProp }) || {}
  const classes = useStyles({ color, visible, skipped: record.skipped })
  const [toggleSkip, loading] = useToggleSkip(resource, record)

  const handleToggleSkip = useCallback(
    (e) => {
      e.preventDefault()
      toggleSkip()
      e.stopPropagation()
    },
    [toggleSkip],
  )

  return (
    <Button
      onClick={handleToggleSkip}
      size={'small'}
      disabled={disabled || loading || record.missing}
      className={clsx(classes.skip, className)}
      title={
        isDateSet(record.skippedAt)
          ? new Date(record.skippedAt).toLocaleString()
          : undefined
      }
      {...rest}
    >
      <BlockIcon fontSize={size} />
    </Button>
  )
}

SkipButton.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object,
  visible: PropTypes.bool,
  color: PropTypes.string,
  size: PropTypes.string,
  component: PropTypes.object,
  disabled: PropTypes.bool,
}

SkipButton.defaultProps = {
  addLabel: true,
  visible: true,
  size: 'small',
  color: 'inherit',
  component: IconButton,
  disabled: false,
}
