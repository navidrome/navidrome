import { useState } from 'react'
import { useSelector } from 'react-redux'

const namesId = (resources, id) =>
  Object.values(resources || {}).some(
    (ids) => Array.isArray(ids) && ids.includes(id),
  )

// Artwork can change behind an unchanged URL (a background conversion replacing its stand-in), so
// this returns when the server last announced new artwork for id, for useImageUrl to version by.
export const useArtworkRefresh = (id) => {
  const refresh = useSelector((state) => state.activity?.refresh)
  const [refreshedAt, setRefreshedAt] = useState()
  if (
    id &&
    refresh &&
    namesId(refresh.resources, id) &&
    refresh.lastReceived !== refreshedAt
  ) {
    setRefreshedAt(refresh.lastReceived)
  }
  return refreshedAt
}
