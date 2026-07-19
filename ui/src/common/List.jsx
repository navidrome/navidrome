import React from 'react'
import { List as RAList } from 'react-admin'
import config from '../config'
import { Pagination } from './Pagination'
import { defaultRowsPerPageOptions, getStoredPerPage } from './perPageStore'
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
      perPage={getStoredPerPage(resource, defaultRowsPerPageOptions)}
      pagination={<Pagination />}
      {...props}
    />
  )
}
