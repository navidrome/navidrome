import React from 'react'
import { docsUrl } from '../utils'

export const DocLink = ({ path, children }) => (
  <a href={docsUrl(path)} target={'_blank'} rel="noopener noreferrer">
    {children}
  </a>
)
