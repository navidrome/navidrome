import React, { useCallback } from 'react'
import { useDispatch } from 'react-redux'
import { Logout } from 'react-admin'
import { clearQueue } from '../audioplayer'

export default (props) => {
  const dispatch = useDispatch()
  const handleClick = useCallback(() => dispatch(clearQueue()), [dispatch])

  return (
    <span onClick={handleClick}>
      <Logout {...props} />
    </span>
  )
}
