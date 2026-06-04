import React from 'react'
import PropTypes from 'prop-types'
import Rating from '@material-ui/lab/Rating'
import { makeStyles } from '@material-ui/core/styles'
import StarIcon from '@material-ui/icons/Star'
import { useRecordContext } from 'react-admin'
import clsx from 'clsx'

const useStyles = makeStyles({
  rating: {
    color: '#ffb400',
    opacity: 0.6,
  },
})

export const AverageRatingField = ({ className, size, ...rest }) => {
  const record = useRecordContext(rest) || {}
  const classes = useStyles()

  const avg = record.averageRating || 0
  if (avg <= 0) return null

  return (
    <span title={`Avg. Rating: ${avg}`}>
      <Rating
        className={clsx(className, classes.rating)}
        value={avg}
        precision={0.5}
        size={size}
        readOnly
        icon={<StarIcon fontSize="inherit" />}
      />
    </span>
  )
}

AverageRatingField.propTypes = {
  record: PropTypes.object,
  size: PropTypes.string,
}

AverageRatingField.defaultProps = {
  size: 'small',
}