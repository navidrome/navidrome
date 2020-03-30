import { Link } from 'react-admin'
import React from 'react'

export const AlbumLinkField = (props) => (
  <Link
    to={`/album/${props.record.albumId}/show`}
    onClick={(e) => e.stopPropagation()}
  >
    {props.record.album}
  </Link>
)

AlbumLinkField.defaultProps = {
  source: 'albumId',
  addLabel: true
}
