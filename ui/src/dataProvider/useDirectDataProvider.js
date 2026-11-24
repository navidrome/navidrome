import { useContext } from 'react'
import { DataProviderContext } from 'react-admin'

// useDataProvider() drops results during an undo window and on a 401; use this when the caller needs them.
export const useDirectDataProvider = () => useContext(DataProviderContext)
