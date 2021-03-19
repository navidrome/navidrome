import React, { useCallback } from 'react'
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

export const StarRating = ({ record = {}, source, resource }) => {
  const [rate, hoverRating, hover] = useStarRating(resource, record, source)
  const classes = useStyles()

  const handleRating = useCallback(
    (e) => {
      e.preventDefault()
      rate(e.target.value)
      e.stopPropagation()
    },
    [rate]
  )

  return (
    <div>
      {[...Array(5)].map((star, i) => {
        const ratingVal = i + 1

        return (
          <label key={i}>
            <input
              className={classes.radioBtn}
              type="radio"
              name="rating"
              value={ratingVal}
              onClick={handleRating}
            />
            <StarIcon
              className={
                ratingVal <= (hover || record[source])
                  ? classes.rated
                  : classes.unrated + ' ' + classes.star
              }
              onMouseEnter={() => hoverRating(ratingVal)}
              onMouseLeave={() => hoverRating(null)}
            />
          </label>
        )
      })}
    </div>
  )
}
