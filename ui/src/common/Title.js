import React from 'react'

const Title = ({ subTitle }) => {
  return <span>CloudSonic {subTitle ? ` - ${subTitle}` : ''}</span>
}

export default Title
