import { useSelector } from 'react-redux'
import { useState } from 'react'
import { useRefresh } from 'react-admin'

export const useResourceRefresh = (...resources) => {
  const [lastTime, setLastTime] = useState(Date.now())
  const refreshData = useSelector(
    (state) => state.activity?.refresh || { lastTime }
  )
  const refresh = useRefresh()

  const resource = refreshData.resource
  if (refreshData.lastTime > lastTime) {
    if (
      resource === '' ||
      resources.length === 0 ||
      resources.includes(resource)
    ) {
      refresh()
    }
    setLastTime(refreshData.lastTime)
  }
}
