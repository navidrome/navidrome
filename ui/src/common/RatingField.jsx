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

  return (
    <span onClick={(e) => stopPropagation(e)}>
      <Rating
        name={record.id}
        className={clsx(
          className,
          classes.rating,
          rating > 0 ? classes.show : classes.hide,
        )}
        value={rating}
        size={size}
        disabled={record?.missing}
        emptyIcon={<StarBorderIcon fontSize="inherit" />}
        onChange={(e, newValue) => handleRating(e, newValue)}
      />
    </span>
  )
}
RatingField.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object,
  visible: PropTypes.bool,
  size: PropTypes.string,
}

RatingField.defaultProps = {
  visible: true,
  size: 'small',
  color: 'inherit',
}
