import React from 'react'
import { Layout } from 'react-admin'
import Menu from './Menu'
import AppBar from './AppBar'

export default (props) => <Layout {...props} menu={Menu} appBar={AppBar} />
