import React from 'react'
import { List as RAList } from 'react-admin'
import config from '../config'
import { Pagination } from './Pagination'
import { Title } from './index'

export const List = (props) => {
  const { resource } = props
  return (
    <RAList
      title={
        <Title
          subTitle={`resources.${resource}.name`}
          args={{ smart_count: 2 }}
        />
      }
      debounce={config.uiSearchDebounceMs}
      perPage={15}
      pagination={<Pagination />}
      {...props}
    />
  )
}
