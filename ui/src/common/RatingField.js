import React from 'react'
import { StarRating } from './StarRating'
import PropTypes from 'prop-types'

export const RatingField = ({ record = {}, resource, size }) => {
  return <StarRating record={record} resource={resource} size={size} />
}

RatingField.propTypes = {
  record: PropTypes.object.isRequired,
  resource: PropTypes.string,
}

RatingField.defaultProps = {
  record: {},
}
