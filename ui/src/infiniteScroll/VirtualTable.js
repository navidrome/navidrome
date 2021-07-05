import React, { isValidElement, useRef, useCallback } from 'react'
import { withStyles } from '@material-ui/core/styles'
import { TableCell } from '@material-ui/core'
import {
  AutoSizer,
  Column,
  InfiniteLoader,
  Table,
} from 'react-virtualized'
import {
  DatagridHeaderCell,
  DatagridCell,
  useDatagridStyles,
  ExpandRowButton,
} from 'react-admin'
import clsx from 'clsx'

const useStyles = (theme) => ({
  row: {
    display: 'flex',
    alignItems: 'center',
    boxSizing: 'border-box',
    cursor: 'pointer',
  },
  tableCell: {
    display: 'flex',
    alignItems: 'center',
    boxSizing: 'border-box',
    flex: 1,
  },
  scrollBlock: {
    backgroundColor: theme.palette.divider,
    borderRadius: 4,
    height: 10,
    width: '60%',
  },
})

function VirtualTable(props) {
  const {
    loadMoreRows,
    isRowLoaded,
    remoteDataCount,
    rowGetter,
    currentSort,
    setSort,
    rowHeight,
    expand,
    classes,
  } = props

  const datagridClasses = useDatagridStyles()
  const children = React.Children.toArray(props.children)

  const cellRenderer = ({ rowData,  cellData, columnIndex, isScrolling }) => {
    const { basePath, resource } = props
    const field = children[columnIndex]

    if (typeof cellData == 'undefined')
      return (
        <TableCell
          component="div"
          className={classes.tableCell}
          style={{ height: rowHeight }}
        >
          <div className={classes.scrollBlock} />
        </TableCell>
      )

    return (
      <DatagridCell
        component="div"
        className={classes.tableCell}
        style={{ height: rowHeight }}
        field={field}
        record={rowData}
        basePath={basePath}
        resource={resource}
      />
    )
  }

  const expandCellRenderer = () => (
    <TableCell
      padding="none"
      component="div"
      className={clsx(classes.tableCell, datagridClasses.expandIconCell)}
      style={{ height: rowHeight }}
    >
      <ExpandRowButton
        classes={datagridClasses}
        expanded={false}
        // TODO: onClick={/* handleToggleExpand */}
        expandContentId={`row-expand`}
      />
    </TableCell>
  )

  const expandHeaderRenderer = () => (
    <TableCell
      padding="none"
      component="div"
      style={{ height: rowHeight }}
      className={clsx(classes.tableCell, datagridClasses.headerCell)}
    />
  )

  const updateSortCallback = useCallback(
    (event) => {
      event.stopPropagation()
      const newField = event.currentTarget.dataset.field
      const newOrder =
        currentSort.field === newField
          ? currentSort.order === 'ASC'
            ? 'DESC'
            : 'ASC'
          : event.currentTarget.dataset.order

      setSort(newField, newOrder)
    },
    [currentSort.field, currentSort.order, setSort]
  )

  const headerRenderer = ({ columnIndex }) => {
    const { resource } = props
    const field = children[columnIndex]

    return (
      <DatagridHeaderCell
        component="div"
        className={clsx(classes.tableCell, datagridClasses.headerCell)}
        field={field}
        currentSort={currentSort}
        isSorting={
          currentSort.field === (field.props.sortBy || field.props.source)
        }
        key={field.props.source || columnIndex}
        resource={resource}
        updateSort={updateSortCallback}
        style={{ height: rowHeight }}
      />
    )
  }

  const defaultflexGrow = 1.0 / (children.length + !!expand)

  return (
    <InfiniteLoader
      // ref={infiniteLoaderRef}
      isRowLoaded={isRowLoaded}
      loadMoreRows={loadMoreRows}
      rowCount={remoteDataCount}
    >
      {({ onRowsRendered, registerChild }) => (
        <AutoSizer disableHeight>
          {({ width }) => (
            // <WindowScroller>
              // {({ height, isScrolling, scrollTop }) => (
                <Table
                  ref={registerChild}
                  onRowsRendered={onRowsRendered}
                  // isScrolling={isScrolling}
                  // scrollTop={scrollTop}
                  width={width}
                  height={rowHeight * 10}
                  // autoHeight
                  headerHeight={rowHeight}
                  rowHeight={rowHeight}
                  rowGetter={rowGetter}
                  rowCount={remoteDataCount}
                  rowClassName={clsx(classes.row, datagridClasses.row)}
                  onRowClick={props.onRowClick}
                >
                  {expand && (
                    <Column
                      key={'expand'}
                      label={'Expand'}
                      dataKey={null}
                      width={60}
                      cellRenderer={expandCellRenderer}
                      headerRenderer={expandHeaderRenderer}
                    />
                  )}
                  {React.Children.map(children, (c, i) =>
                    isValidElement(c) && c.props ? (
                      <Column
                        key={i}
                        label={c.props.source}
                        dataKey={c.props.source}
                        width={c.props.width || 100}
                        flexGrow={c.props.flexGrow || defaultflexGrow}
                        cellRenderer={(cellRenderProps) =>
                          cellRenderer({ ...cellRenderProps, columnIndex: i })
                        }
                        headerRenderer={(headerProps) =>
                          headerRenderer({ ...headerProps, columnIndex: i })
                        }
                      />
                    ) : null
                  )}
                </Table>
              // )}
            // </WindowScroller>
          )}
        </AutoSizer>
      )}
    </InfiniteLoader>
  )
}

VirtualTable.defaultProps = {
  rowHeight: 52,
}

export default withStyles(useStyles)(VirtualTable)
