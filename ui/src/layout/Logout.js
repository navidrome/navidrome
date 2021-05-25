import React, { useCallback } from 'react'
import { useDispatch } from 'react-redux'
import { Logout as RALogout } from 'react-admin'
import { clearQueue } from '../actions'

const Logout = (props) => {
  const dispatch = useDispatch()
  const handleClick = useCallback(() => dispatch(clearQueue()), [dispatch])

  return (
    <span onClick={handleClick}>
      <RALogout {...props} />
    </span>
  )
}

export default Logout
