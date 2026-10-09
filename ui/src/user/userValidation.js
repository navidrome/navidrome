// User form validation utilities
export const validateUserForm = (values, translate) => {
  const errors = {}

  // Only require library selection for non-admin users
  if (!values.isAdmin) {
    // Check both libraryIds (array of IDs) and libraries (array of objects)
    const hasLibraryIds = values.libraryIds && values.libraryIds.length > 0
    const hasLibraries = values.libraries && values.libraries.length > 0

    if (!hasLibraryIds && !hasLibraries) {
      errors.libraryIds = translate(
        'resources.user.validation.librariesRequired',
      )
    }
  }

  if (values.scrobbleFilter && values.scrobbleFilter.trim() !== '') {
    try {
      JSON.parse(values.scrobbleFilter)
    } catch {
      errors.scrobbleFilter = translate(
        'resources.user.validation.invalidScrobbleFilter',
      )
    }
  }

  return errors
}
