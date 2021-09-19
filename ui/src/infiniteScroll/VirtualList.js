import MUIList from '@material-ui/core/List'
import { useListContext } from 'ra-core'
import { InfiniteLoader, AutoSizer, List } from 'react-virtualized'
import config from '../config'
import useVirtualizedData from './useVirtualizedData'


function MaterialList({ renderItem, ...rest }) {
  const { ids, data } = useListContext()
  return (
    <MUIList
      {...rest}
    >
      {ids.map(id => renderItem(data[id]))}
    </MUIList>
  )
}

function VirtualList({ renderItem, itemHeight }) {
  const { data, loadedIds, total, handleLoadMore } = useVirtualizedData()

  const rowCount = total || 0

  return (
    <InfiniteLoader
      isRowLoaded={({ index }) => !!loadedIds[index]}
      loadMoreRows={handleLoadMore}
      rowCount={rowCount}
    >
      {({ onRowsRendered, registerChild }) => (
        <AutoSizer>
          {({ width, height }) => (
            <List
              ref={registerChild}
              rowHeight={itemHeight}
              height={height}
              width={width}
              onRowsRendered={onRowsRendered}
              rowRenderer={({ index, style, key }) => (
                <div style={{ ...style, listStyleType: 'none' }} key={key}>
                  {renderItem(data[loadedIds[index]])}
                </div>
              )}
              rowCount={rowCount}
            />
          )}
        </AutoSizer>
      )}
    </InfiniteLoader>
  )
}

VirtualList.defaultProps = {
  itemHeight: 100,
  renderItem: () => null,
}

export default (config.enableInfiniteScroll ? VirtualList : MaterialList)
