import { useState, useEffect } from 'react';
import { List, useQuery } from 'react-admin'

function InfiniteList({ resource, children, perPage, ...rest }) {
  const [loadedRowCount, setLoadedRowCount] = useState(0);
  const [loadedRowsMap, setLoadedRowsMap] = useState({});

  const isRowLoaded = ({ index }) => !!loadedRowsMap[index];

  const [loadPromiseResolver, setLoadPromiseResolver] = useState(null)
  useEffect(() => {
    if (loadPromiseResolver) {
      loadPromiseResolver()
      setLoadPromiseResolver(null)
    }

  }, [loadedRowCount]);

  const loadMoreRows = ({ startIndex, stopIndex }) => {
    const increment = stopIndex - startIndex + 1;
    const page = startIndex/perPage + 1;
    const { data, total, loading, error } = useQuery({
      type: 'getList',
      resource,
      payload: {
          pagination: { page, perPage },
          sort,
          filter: {},
      }
    });

    return new Promise((resolve) => {
      setLoadPromiseResolver(resolve);
    })
  }

  return children;
}

export default InfiniteList
