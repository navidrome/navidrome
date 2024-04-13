import React, { useCallback } from 'react'
import PropTypes from 'prop-types'
import Rating from '@material-ui/lab/Rating'
import { makeStyles } from '@material-ui/core/styles'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import clsx from 'clsx'
import { useRating } from './useRating'
import { useRecordContext } from 'react-admin'

const useStyles = makeStyles({
  rating: {
    color: (props) => props.color,
    visibility: (props) => (props.visible === false ? 'hidden' : 'inherit'),
  },
  show: {
    visibility: 'visible !important',
  },
  hide: {
    visibility: 'hidden',
  },
})

export const RatingField = ({
  resource,
  visible,
  className,
  size,
  color,
  disabled,
  ...rest
}) => {
  const record = useRecordContext(rest) || {}
  const [rate, rating] = useRating(resource, record)
  const classes = useStyles({ color, visible })

  const stopPropagation = (e) => {
    e.stopPropagation()
  }

  const handleRating = useCallback(
    (e, val) => {
      rate(val ?? 0, e.target.name)
    },
    [rate],
  )

  // If the rating is not yet loaded or unset, it will be `undefined`.
  // material-ui uses an `null` to indicate an unset value.
  // Passing `undefined` will switch the component into "uncontrolled" mode.
  const ratingValue = rating ?? null

  return (
    <span onClick={(e) => stopPropagation(e)}>
      <Rating
        name={record.id}
        className={clsx(
          className,
          classes.rating,
          rating > 0 ? classes.show : classes.hide,
        )}
        value={ratingValue}
        size={size}
        emptyIcon={<StarBorderIcon fontSize="inherit" />}
        onChange={(e, newValue) => handleRating(e, newValue)}
        disabled={disabled}
      />
    </span>
  )
}
RatingField.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object,
  visible: PropTypes.bool,
  disabled: PropTypes.bool,
  size: PropTypes.string,
}

RatingField.defaultProps = {
  visible: true,
  size: 'small',
  color: 'inherit',
  disabled: false,
}
