import { useState } from 'react'
import { useSelector } from 'react-redux'

const namesId = (resources, id) =>
  Object.values(resources || {}).some(
    (ids) => Array.isArray(ids) && ids.includes(id),
  )

// Artwork can change behind an unchanged URL (a background conversion replacing its stand-in), so
// this returns when the server last announced new artwork for id, for useImageUrl to version by.
export const useArtworkRefresh = (id) => {
  // A primitive, so an event naming other items does not re-render this cover.
  const named = useSelector(({ activity }) =>
    id && namesId(activity?.refresh?.resources, id)
      ? activity.refresh.lastReceived
      : undefined,
  )
  const [refreshedAt, setRefreshedAt] = useState()
  if (named !== undefined && named !== refreshedAt) {
    setRefreshedAt(named)
  }
  return refreshedAt
}
