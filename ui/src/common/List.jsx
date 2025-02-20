import React from 'react'
import { List as RAList } from 'react-admin'
import { Pagination } from './Pagination'
import { Title } from './index'
import { defaultPageSize } from '../utils/pageSizes'

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
      perPage={defaultPageSize()}
      pagination={<Pagination />}
      {...props}
    />
  )
}
