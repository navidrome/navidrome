import { useSelector } from 'react-redux'
import { useState } from 'react'
import { useRefresh, useDataProvider } from 'react-admin'

export const useResourceRefresh = (...visibleResources) => {
  const [lastTime, setLastTime] = useState(Date.now())
  const refresh = useRefresh()
  const dataProvider = useDataProvider()
  const refreshData = useSelector(
    (state) => state.activity?.refresh || { lastReceived: lastTime },
  )
  const { resources, lastReceived } = refreshData

  if (lastReceived <= lastTime) {
    return
  }
  setLastTime(lastReceived)

  if (
    resources &&
    (resources['*'] === '*' ||
      Object.values(resources).find((v) => v.find((v2) => v2 === '*')))
  ) {
    refresh()
    return
  }
  if (resources) {
    Object.keys(resources).forEach((r) => {
      if (visibleResources.length === 0 || visibleResources?.includes(r)) {
        if (resources[r]?.length > 0) {
          dataProvider.getMany(r, { ids: resources[r] })
        }
      }
    })
  }
}
