import React, {
  createContext,
  useContext,
  useEffect,
  useRef,
  useCallback,
  useState,
  useMemo,
} from 'react'
import { useListContext, useGetList } from 'react-admin'
import { useHistory, useLocation } from 'react-router-dom'

const InfiniteScrollContext = createContext(null)

const scrollStateCache = new Map()

const getStorageKey = (pathname, filterValues) => {
  const filterKey = JSON.stringify(filterValues || {})
  return `${pathname}:${filterKey}`
}

const saveScrollState = (pathname, filterValues, pagesLoaded, scrollY) => {
  const key = getStorageKey(pathname, filterValues)
  scrollStateCache.set(key, { pagesLoaded, scrollY })
}

const getScrollState = (pathname, filterValues) => {
  const key = getStorageKey(pathname, filterValues)
  return scrollStateCache.get(key) || null
}

const clearScrollState = (pathname, filterValues) => {
  const key = getStorageKey(pathname, filterValues)
  scrollStateCache.delete(key)
}

export const InfiniteScrollProvider = ({ children }) => {
  const history = useHistory()
  const location = useLocation()

  const {
    data: page1Data,
    ids: page1Ids,
    total,
    perPage,
    loading: page1Loading,
    loaded: page1Loaded,
    resource,
    filterValues,
    sort,
    filter,
  } = useListContext()

  const [nextPageToFetch, setNextPageToFetch] = useState(null)
  const [additionalPagesData, setAdditionalPagesData] = useState({})
  const [isFetchingMore, setIsFetchingMore] = useState(false)
  const [isRestoring, setIsRestoring] = useState(false)
  const [targetPage, setTargetPage] = useState(null)
  const [pendingScrollY, setPendingScrollY] = useState(null)

  const initializedRef = useRef(false)
  const navigationTypeRef = useRef(null)
  const sentinelRef = useRef(null)

  const combinedFilter = useMemo(
    () => ({
      ...filter,
      ...filterValues,
    }),
    [filter, filterValues],
  )

  useEffect(() => {
    if (typeof window !== 'undefined' && window.performance) {
      const navEntries = window.performance.getEntriesByType('navigation')
      if (navEntries.length > 0) {
        navigationTypeRef.current = navEntries[0].type
      }
    }
  }, [])

  const {
    data: additionalData,
    ids: additionalIds,
    loaded: additionalLoaded,
  } = useGetList(
    resource,
    { page: nextPageToFetch || 1, perPage },
    sort,
    combinedFilter,
    { enabled: nextPageToFetch !== null && nextPageToFetch > 1 },
  )

  const maxLoadedPage = useMemo(() => {
    const additionalPages = Object.keys(additionalPagesData).map(Number)
    if (additionalPages.length > 0) {
      return Math.max(...additionalPages)
    }
    return page1Loaded ? 1 : 0
  }, [additionalPagesData, page1Loaded])

  const hasNextPage = useMemo(() => {
    if (!total || !perPage) return false
    const totalPages = Math.ceil(total / perPage)
    return maxLoadedPage < totalPages
  }, [total, perPage, maxLoadedPage])

  const { accumulatedData, accumulatedIds } = useMemo(() => {
    const accData = { ...page1Data }
    const accIds = [...(page1Ids || [])]
    const seenIds = new Set(accIds)

    const sortedPages = Object.keys(additionalPagesData)
      .map(Number)
      .sort((a, b) => a - b)

    for (const pageNum of sortedPages) {
      const pageInfo = additionalPagesData[pageNum]
      if (pageInfo) {
        Object.assign(accData, pageInfo.data)
        for (const id of pageInfo.ids) {
          if (!seenIds.has(id)) {
            seenIds.add(id)
            accIds.push(id)
          }
        }
      }
    }

    return { accumulatedData: accData, accumulatedIds: accIds }
  }, [page1Data, page1Ids, additionalPagesData])

  useEffect(() => {
    if (initializedRef.current || !page1Loaded) return
    initializedRef.current = true

    const isBackNavigation =
      history.action === 'POP' || navigationTypeRef.current === 'back_forward'

    if (isBackNavigation) {
      const saved = getScrollState(location.pathname, filterValues)
      if (saved && saved.pagesLoaded > 1) {
        setIsRestoring(true)
        setTargetPage(saved.pagesLoaded)
        if (saved.scrollY > 0) {
          setPendingScrollY(saved.scrollY)
        }
        setNextPageToFetch(2)
        return
      }
    }

    clearScrollState(location.pathname, filterValues)
    window.scrollTo(0, 0)
  }, [page1Loaded, history.action, location.pathname, filterValues])

  useEffect(() => {
    if (
      nextPageToFetch &&
      nextPageToFetch > 1 &&
      additionalLoaded &&
      additionalData &&
      additionalIds
    ) {
      setAdditionalPagesData((prev) => {
        if (!prev[nextPageToFetch]) {
          return {
            ...prev,
            [nextPageToFetch]: {
              data: { ...additionalData },
              ids: [...additionalIds],
            },
          }
        }
        return prev
      })

      setIsFetchingMore(false)

      if (isRestoring && targetPage && nextPageToFetch < targetPage) {
        setNextPageToFetch(nextPageToFetch + 1)
      } else if (isRestoring && targetPage && nextPageToFetch >= targetPage) {
        if (pendingScrollY) {
          setTimeout(() => {
            window.scrollTo(0, pendingScrollY)
            setPendingScrollY(null)
            setIsRestoring(false)
            setTargetPage(null)
          }, 100)
        } else {
          setIsRestoring(false)
          setTargetPage(null)
        }
      }
    }
  }, [
    nextPageToFetch,
    additionalLoaded,
    additionalData,
    additionalIds,
    isRestoring,
    targetPage,
    pendingScrollY,
  ])

  useEffect(() => {
    if (!isRestoring && maxLoadedPage > 0 && page1Loaded) {
      const handleScroll = () => {
        saveScrollState(
          location.pathname,
          filterValues,
          maxLoadedPage,
          window.scrollY,
        )
      }

      handleScroll()

      let scrollTimeout
      const throttledScroll = () => {
        clearTimeout(scrollTimeout)
        scrollTimeout = setTimeout(handleScroll, 150)
      }

      window.addEventListener('scroll', throttledScroll, { passive: true })
      return () => {
        window.removeEventListener('scroll', throttledScroll)
        clearTimeout(scrollTimeout)
      }
    }
  }, [maxLoadedPage, isRestoring, location.pathname, filterValues, page1Loaded])

  const loadMore = useCallback(() => {
    if (!page1Loading && !isFetchingMore && hasNextPage && !isRestoring) {
      setIsFetchingMore(true)
      setNextPageToFetch(maxLoadedPage + 1)
    }
  }, [page1Loading, isFetchingMore, hasNextPage, maxLoadedPage, isRestoring])

  useEffect(() => {
    const sentinel = sentinelRef.current
    if (!sentinel) return

    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[0]
        if (
          entry.isIntersecting &&
          hasNextPage &&
          !page1Loading &&
          !isFetchingMore &&
          !isRestoring
        ) {
          loadMore()
        }
      },
      {
        root: null,
        rootMargin: '200px',
        threshold: 0,
      },
    )

    observer.observe(sentinel)

    return () => {
      observer.disconnect()
    }
  }, [hasNextPage, page1Loading, isFetchingMore, loadMore, isRestoring])

  const value = useMemo(
    () => ({
      accumulatedData,
      accumulatedIds,
      hasNextPage,
      isFetchingMore,
      loading: page1Loading || isFetchingMore || isRestoring,
      sentinelRef,
      total,
      loaded: page1Loaded && !isRestoring,
      loadMore,
    }),
    [
      accumulatedData,
      accumulatedIds,
      hasNextPage,
      isFetchingMore,
      page1Loading,
      isRestoring,
      total,
      page1Loaded,
      loadMore,
    ],
  )

  return (
    <InfiniteScrollContext.Provider value={value}>
      {children}
    </InfiniteScrollContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export const useInfiniteScroll = () => {
  const context = useContext(InfiniteScrollContext)
  if (!context) {
    throw new Error(
      'useInfiniteScroll must be used within an InfiniteScrollProvider',
    )
  }
  return context
}
