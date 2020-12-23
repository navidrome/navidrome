import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-admin'

export const AlbumLinkField = (props) => (
  <Link
    to={`/album/${props.record.albumId}/show`}
    onClick={(e) => e.stopPropagation()}
  >
    {props.record.album}
  </Link>
)

AlbumLinkField.propTypes = {
  sortBy: PropTypes.string,
  sortByOrder: PropTypes.oneOf(['ASC', 'DESC']),
}

AlbumLinkField.defaultProps = {
  addLabel: true,
}
