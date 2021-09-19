import { List as RAList } from '../common'
import { Datagrid as RADatagrid } from 'ra-ui-materialui'

import InfiniteDatagrid from './Datagrid'
import InfiniteList from './List'
import config from '../config'

export const List = config.enableInfiniteScroll ? InfiniteList : RAList
export const Datagrid = config.enableInfiniteScroll
  ? InfiniteDatagrid
  : RADatagrid
