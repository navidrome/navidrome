import React from 'react'
import { useSelector } from 'react-redux'
import { CircularProgress, Box } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { InfiniteScrollProvider, useInfiniteScroll } from './useInfiniteScroll'

const useStyles = makeStyles((theme) => ({
  sentinel: {
    width: '100%',
    height: '1px',
    marginTop: theme.spacing(2),
  },
  loadingContainer: {
    display: 'flex',
    justifyContent: 'center',
    padding: theme.spacing(2),
    width: '100%',
  },
}))

const InfiniteScrollContent = ({ children, ...props }) => {
  const classes = useStyles()
  const {
    accumulatedData,
    accumulatedIds,
    loading,
    loaded,
    hasNextPage,
    isFetchingMore,
    sentinelRef,
  } = useInfiniteScroll()

  const infiniteProps = {
    ...props,
    data: accumulatedData,
    ids: accumulatedIds,
    loading,
    loaded,
  }

  return (
    <>
      {React.Children.map(children, (child) =>
        React.cloneElement(child, infiniteProps),
      )}
      {isFetchingMore && (
        <Box className={classes.loadingContainer}>
          <CircularProgress size={24} />
        </Box>
      )}
      {hasNextPage && <div ref={sentinelRef} className={classes.sentinel} />}
    </>
  )
}

export const InfiniteScrollWrapper = ({ children, ...props }) => {
  const infiniteScrollEnabled = useSelector(
    (state) => state.infiniteScroll?.enabled ?? false,
  )

  if (!infiniteScrollEnabled) {
    return React.Children.map(children, (child) =>
      React.cloneElement(child, props),
    )
  }

  return (
    <InfiniteScrollProvider>
      <InfiniteScrollContent {...props}>{children}</InfiniteScrollContent>
    </InfiniteScrollProvider>
  )
}
