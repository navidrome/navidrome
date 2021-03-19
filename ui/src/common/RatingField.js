import React from 'react'
import { StarRating } from './StarRating'

export const RatingField = ({ record = {}, source, resource }) => {
  return (
    <div>
      <StarRating record={record} source={source} resource={resource} />
    </div>
  )
}
