import React from 'react'
import { docsUrl } from '../utils/docsUrl'

const DocLink = ({ path, children }) => (
  <a href={docsUrl(path)} target={'_blank'} rel="noopener noreferrer">
    {children}
  </a>
)

export default DocLink
