import { useSelector } from 'react-redux'

/**
 * Hook to get the currently selected library IDs
 * Returns an array of library IDs that should be used for filtering data
 * If no libraries are selected (empty array), returns all user accessible libraries
 */
export const useSelectedLibraries = () => {
  const { userLibraries, selectedLibraries } = useSelector(
    (state) => state.library,
  )

  // If no specific selection, default to all accessible libraries
  if (selectedLibraries.length === 0 && userLibraries.length > 0) {
    return userLibraries.map((lib) => lib.id)
  }

  return selectedLibraries
}

/**
 * Hook to get library filter parameters for data provider queries
 * Returns an object that can be spread into query parameters
 */
export const useLibraryFilter = () => {
  const selectedLibraryIds = useSelectedLibraries()

  // If user has access to only one library or no specific selection, no filter needed
  if (selectedLibraryIds.length <= 1) {
    return {}
  }

  return {
    libraryIds: selectedLibraryIds,
  }
}

/**
 * Hook to check if a specific library is currently selected
 */
export const useIsLibrarySelected = (libraryId) => {
  const selectedLibraryIds = useSelectedLibraries()
  return selectedLibraryIds.includes(libraryId)
}
