import { InfiniteLoader, AutoSizer, List } from 'react-virtualized'
import { useInstance } from './useInstance'
import { useEffect } from 'react'
import { useSelector, useDispatch } from 'react-redux'
import { useListContext, crudGetList } from 'ra-core'
import { GridList } from '@material-ui/core'

function AlbumDatagrid(props) {
  const { children, tileHeight, columns } = props

  const { resource, perPage, currentSort, filterValues } = useListContext()

  const { data, ids, total } = useSelector((state) => ({
    ids: state.admin.resources[resource].list.ids,
    data: state.admin.resources[resource].data,
    total: state.admin.resources[resource].list.total,
    loadedOnce: state.admin.resources[resource].list.loadedOnce,
  }))
  const dispatch = useDispatch()

  const [loadedIds, updateLoadedIds] = useInstance({})

  const getList = (...args) => dispatch(crudGetList(...args))

  const [lastFetchPosition, updateLastFetchPosition] = useInstance({
    startIndex: 0,
    stopIndex: perPage,
  })

  const [loadPromiseResolver, updateLoadPromiseResolver] = useInstance(null)

  const handleLoadMore = (query) => {
    const startIndex = getIndexesForRow(query.startIndex)[0]
    const stopIndex = getIndexesForRow(query.stopIndex).pop()
    console.log('LoadMore', startIndex, stopIndex)
    const page = Math.floor(startIndex / perPage) + 1
    const newStopIndex = Math.min(total, stopIndex + perPage - 1)

    return new Promise((resolve) => {
      updateLoadPromiseResolver(resolve)
      updateLastFetchPosition({ startIndex, stopIndex: newStopIndex })
      getList(resource, { page: page, perPage }, currentSort, filterValues)
    })
  }

  useEffect(() => {
    let { startIndex, stopIndex } = lastFetchPosition
    let newLoadedIds = loadedIds

    if (loadPromiseResolver == null) {
      startIndex = 0
      stopIndex = perPage
      newLoadedIds = {}
      // TODO: scrollToPosition(0)
    }

    for (let i = startIndex; i <= stopIndex; i++) {
      newLoadedIds[i] = ids[i - startIndex]
    }

    updateLoadedIds(newLoadedIds)
    updateLastFetchPosition({ startIndex, stopIndex })

    if (loadPromiseResolver) {
      loadPromiseResolver()
      updateLoadPromiseResolver(null)
    }
  }, [ids])

  const getIndexesForRow = (index) => {
    const res = []
    for (let i = 0; i < columns; i++) {
      if (columns * index + i === total - 1) break
      res.push(columns * index + i)
    }
    return res
  }

  const rowRenderer = ({ index, style, key }) => {
    const isLoaded = isRowLoaded({ index })
    const itemsForRow = getIndexesForRow(index, columns)

    return (
      <div style={style} key={key}>
        <GridList
          component={'div'}
          cellHeight={'auto'}
          cols={columns}
          spacing={20}
        >
          {itemsForRow.map((itemIndex) =>
            children({
              isLoaded: isLoaded && data[loadedIds[itemIndex]],
              record: data[loadedIds[itemIndex]],
              index: itemIndex,
            })
          )}
        </GridList>
      </div>
    )
  }

  const isRowLoaded = ({ index }) => {
    const indices = getIndexesForRow(index, columns)
    return indices.reduce((prev, curr) => prev && loadedIds[curr], true)
  }

  const rowCount = Math.ceil(total / columns)

  return (
    <InfiniteLoader
      isRowLoaded={isRowLoaded}
      loadMoreRows={handleLoadMore}
      rowCount={rowCount}
    >
      {({ onRowsRendered, registerChild }) => (
        <AutoSizer disableHeight>
          {({ width }) => (
            <List
              ref={registerChild}
              rowHeight={tileHeight}
              height={tileHeight * 2}
              width={width}
              onRowsRendered={onRowsRendered}
              rowRenderer={rowRenderer}
              rowCount={rowCount}
            />
          )}
        </AutoSizer>
      )}
    </InfiniteLoader>
  )
}

AlbumDatagrid.defaultProps = {
  tileHeight: 245,
}
export default AlbumDatagrid
