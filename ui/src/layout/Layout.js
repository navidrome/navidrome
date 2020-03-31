import React from 'react'
import { useSelector } from 'react-redux'
import { Layout } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import Menu from './Menu'
import AppBar from './AppBar'
import { DarkTheme, LightTheme } from '../themes'

const useStyles = makeStyles({
  root: { paddingBottom: '80px' }
})

export default (props) => {
  const classes = useStyles()
  const theme = useSelector((state) =>
    state.theme === 'dark' ? DarkTheme : LightTheme
  )

  return (
    <>
      <Layout
        {...props}
        className={classes.root}
        menu={Menu}
        appBar={AppBar}
        theme={theme}
      />
    </>
  )
}
