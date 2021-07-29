import { InfiniteLoader, AutoSizer, List } from 'react-virtualized'
import { GridList } from '@material-ui/core'
import useVirtualizedData from './useVirtualizedData'

function AlbumDatagrid(props) {
  const { children, itemHeight, columns } = props
  const { data, loadedIds, total, handleLoadMore } = useVirtualizedData()

  const loadMoreRows = (query) => {
    const startIndex = getIndexesForRow(query.startIndex)[0]
    const stopIndex = getIndexesForRow(query.stopIndex).pop()
    return handleLoadMore({ startIndex, stopIndex })
  }

  const getIndexesForRow = (index) => {
    const res = []
    for (let i = 0; i < columns; i++) {
      if (columns * index + i === total - 1) break
      res.push(columns * index + i)
    }
    return res
  }

  // For react-virtualized, the number of rows is total/columns, since each row
  // displays "column" no. of items
  const rowCount = Math.ceil(total / columns)

  const isRowLoaded = ({ index }) => {
    const indices = getIndexesForRow(index, columns)
    return indices.reduce((prev, curr) => prev && loadedIds[curr], true)
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

  return (
    <InfiniteLoader
      isRowLoaded={isRowLoaded}
      loadMoreRows={loadMoreRows}
      rowCount={rowCount}
    >
      {({ onRowsRendered, registerChild }) => (
        <AutoSizer disableHeight>
          {({ width }) => (
            <List
              ref={registerChild}
              rowHeight={itemHeight}
              height={itemHeight * 2}
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
  itemHeight: 245,
  columns: 3,
  children: () => null,
}

export default AlbumDatagrid
