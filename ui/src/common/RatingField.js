import React from 'react'
import { StarRating } from './StarRating'
import PropTypes from 'prop-types'

export const RatingField = ({ record = {}, resource }) => {
  return <StarRating record={record} resource={resource} />
}

RatingField.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object.isRequired,
}

RatingField.defaultProps = {
  record: {},
}
