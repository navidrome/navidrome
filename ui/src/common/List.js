import React from 'react'
import { List as RAList } from 'react-admin'
import { Pagination } from './Pagination'
import { Title } from './index'

export const LIST_PER_PAGE_DEFAULT = 15

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
      perPage={LIST_PER_PAGE_DEFAULT}
      pagination={<Pagination />}
      {...props}
    />
  )
}
