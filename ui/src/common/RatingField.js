import React, { useCallback } from 'react'
import PropTypes from 'prop-types'
import Rating from '@material-ui/lab/Rating'
import { makeStyles } from '@material-ui/core/styles'
import clsx from 'clsx'
import { useRating } from './useRating'

const useStyles = makeStyles({
  rating: {
    visibility: (props) =>
      props.visible === false
        ? 'hidden'
        : props.rating > 0
        ? 'visible !important'
        : 'inherit',
  },
})

export const RatingField = ({ resource, record, visible, className, size }) => {
  const classes = useStyles({ visible, rating: record.rating })
  const [rate] = useRating(resource, record)

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
        className={clsx(className, classes.rating)}
        value={record.rating}
        size={size}
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
  size: 'medium',
}
