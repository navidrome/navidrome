import React from 'react'
import { useSelector } from 'react-redux'
import { Layout } from 'react-admin'
import Menu from './Menu'
import AppBar from './AppBar'
import { DarkTheme, LightTheme } from '../themes'

export default (props) => {
  const theme = useSelector((state) =>
    state.theme === 'dark' ? DarkTheme : LightTheme
  )

  return <Layout {...props} menu={Menu} appBar={AppBar} theme={theme} />
}
