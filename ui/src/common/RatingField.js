import React, { useCallback } from 'react'
import PropTypes from 'prop-types'
import Rating from '@material-ui/lab/Rating'
import { makeStyles } from '@material-ui/core/styles'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import clsx from 'clsx'
import { useRating } from './useRating'

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
  record,
  visible,
  className,
  size,
  color,
}) => {
  const [rate, rating] = useRating(resource, record)
  const classes = useStyles({ visible, rating: record.rating, color })

  const stopPropagation = (e) => {
    e.stopPropagation()
  }

  const handleRating = useCallback(
    (e, val) => {
      rate(val, e.target.name)
    },
    [rate]
  )

  return (
    <span onClick={(e) => stopPropagation(e)}>
      <Rating
        name={record.id}
        className={clsx(
          className,
          classes.rating,
          rating > 0 ? classes.show : classes.hide
        )}
        value={rating}
        size={size}
        emptyIcon={<StarBorderIcon fontSize="inherit" />}
        onChange={(e, newValue) => handleRating(e, newValue)}
      />
    </span>
  )
}
RatingField.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object.isRequired,
  visible: PropTypes.bool,
  size: PropTypes.string,
}

RatingField.defaultProps = {
  record: {},
  visible: true,
  size: 'small',
  color: 'inherit',
}
