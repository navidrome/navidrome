import React, { useCallback } from 'react'
import PropTypes from 'prop-types'
import StarIcon from '@material-ui/icons/Star'
import { makeStyles } from '@material-ui/core/styles'
import { useStarRating } from './useStarRating'

const useStyles = makeStyles({
  rated: {
    color: '#ffc107',
  },
  unrated: {
    color: '#e4e5e9',
  },
  radioBtn: {
    opacity: 0,
    width: 0,
    height: 0,
  },
  star: {
    cursor: 'pointer',
    transition: 'color 200ms',
  },
})

export const StarRating = ({ record = {}, resource }) => {
  const [rate, hoverRating, hover] = useStarRating(resource, record)
  const classes = useStyles()
  const handleRating = useCallback(
    (e, ratingVal) => {
      e.preventDefault()
      rate(ratingVal)
      e.stopPropagation()
    },
    [rate]
  )

  const handleHover = (ratingVal) => {
    hoverRating(ratingVal)
  }

  // const stopPropagation = (e, ratingVal) => {
  //   e.preventDefault()
  //   console.log('calling rate()')
  //   rate(ratingVal)
  //   e.stopPropagation()
  // }

  return (
    <>
      {[...Array(5)].map((star, i) => {
        const ratingVal = i + 1

        return (
          <label key={i} onClick={(e) => handleRating(e, ratingVal)}>
            <input
              className={classes.radioBtn}
              type="radio"
              name="rating"
              value={ratingVal}
            />
            <StarIcon
              className={
                classes.star +
                ' ' +
                (ratingVal <= (hover || record.rating)
                  ? classes.rated
                  : classes.unrated)
              }
              onMouseEnter={() => handleHover(ratingVal)}
              onMouseLeave={() => handleHover(null)}
            />
          </label>
        )
      })}
    </>
  )
}

StarRating.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object.isRequired,
}

StarRating.defaultProps = {
  record: {},
}
