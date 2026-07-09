import React, { useEffect, useState } from 'react'
import {
  List,
  Datagrid,
  useTranslate,
  FunctionField,
  useDataProvider,
  Pagination,
} from 'react-admin'
import FolderIcon from '@material-ui/icons/Folder'
import { makeStyles } from '@material-ui/core'
import { useSelector } from 'react-redux'
import { Title, FolderContextMenu } from '../common'
import FolderListActions from './FolderListActions'
import FolderGridView from './FolderGridView'

const useStyles = makeStyles({
  icon: {
    verticalAlign: 'middle',
    marginRight: '10px',
  },
  row: {
    '&:hover': {
      '& $contextMenu': {
        visibility: 'visible',
      },
    },
  },
  contextMenu: {
    visibility: 'hidden',
  },
})

const FolderList = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const folderView = useSelector((state) => state.folderView)
  const [libraryId, setLibraryId] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    dataProvider
      .getList('library', {
        pagination: { page: 1, perPage: 1 },
        sort: { field: 'id', order: 'ASC' },
        filter: {},
      })
      .then(({ data }) => {
        if (data.length > 0) {
          setLibraryId(data[0].id.toString())
        }
        setLoading(false)
      })
      .catch(() => setLoading(false))
  }, [dataProvider])

  if (loading) return null
  if (!libraryId)
    return (
      <div style={{ padding: '20px' }}>
        No libraries found. Please scan your music first.
      </div>
    )

  return (
    <List
      {...props}
      perPage={500}
      sort={{ field: 'name', order: 'ASC' }}
      filter={{ parent_id: libraryId }}
      actions={<FolderListActions />}
      pagination={<Pagination rowsPerPageOptions={[100, 250, 500, 1000]} />}
      title={<Title title={translate('menu.folders')} />}
    >
      {folderView.grid ? (
        <FolderGridView {...props} />
      ) : (
        <Datagrid rowClick="show" classes={{ row: classes.row }}>
          <FunctionField
            source="name"
            render={(record) => {
              if (!record || !record.name) return null
              return (
                <>
                  <FolderIcon className={classes.icon} />
                  {record.name}
                </>
              )
            }}
          />
          <FolderContextMenu
            source="name"
            className={classes.contextMenu}
            showLove={false}
          />
        </Datagrid>
      )}
    </List>
  )
}

export default FolderList
