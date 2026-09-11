import { useCallback } from 'react'
import { useDataProvider, useNotify } from 'react-admin'

export const useRefreshMetadata = () => {
  const dataProvider = useDataProvider()
  const notify = useNotify()

  return useCallback(
    (resource, id) =>
      dataProvider
        .refreshMetadata(resource, id)
        .then(() => notify('message.metadataRefreshStarted'))
        .catch(() => notify('ra.page.error', 'warning')),
    [dataProvider, notify],
  )
}
