import React from 'react'

const Title = ({ subTitle }) => {
  return <span>Navidrome {subTitle ? ` - ${subTitle}` : ''}</span>
}

export default Title
