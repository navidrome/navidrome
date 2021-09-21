import { List as RAList } from '../common'
import { Datagrid as RADatagrid } from 'ra-ui-materialui'

import InfiniteDatagrid from './Datagrid'
import InfiniteList from './List'
import config from '../config'

export const List = config.devEnableInfiniteScroll ? InfiniteList : RAList
export const Datagrid = config.devEnableInfiniteScroll
  ? InfiniteDatagrid
  : RADatagrid
