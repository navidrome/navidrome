import type { Record } from 'ra-core'

declare module 'react-redux' {
  interface DefaultRootState {
    moveToIndexDialog: {
      open: boolean
      record?: Record
    }
  }
}
