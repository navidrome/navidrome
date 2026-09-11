import React from 'react'
import Link from '@material-ui/core/Link'
import { docsUrl } from '../utils'

export const DocLink = ({ path, children, ...rest }) => (
  <Link
    href={docsUrl(path)}
    target="_blank"
    rel="noopener noreferrer"
    {...rest}
  >
    {children}
  </Link>
)
