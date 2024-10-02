import React from 'react'
import { Provider } from 'react-redux'
import { createStore } from 'redux'
import mediaQuery from 'css-mediaquery'
import { renderHook } from '@testing-library/react-hooks'
import useCurrentTheme from './useCurrentTheme'
import { themeReducer } from '../reducers/themeReducer'
import { AUTO_THEME_ID } from '../consts'

function createMatchMedia(theme) {
  return (query) => ({
    matches: mediaQuery.match(query, { 'prefers-color-scheme': theme }),
    addListener: () => {},
    removeListener: () => {},
  })
}

describe('useCurrentTheme', () => {
  describe('with user preference theme as light', () => {
    beforeAll(() => {
      window.matchMedia = createMatchMedia('light')
    })
    it('sets theme as light in auto mode', () => {
      const { result } = renderHook(() => useCurrentTheme(), {
        wrapper: ({ children }) => (
          <Provider store={createStore(themeReducer, { theme: AUTO_THEME_ID })}>
            {children}
          </Provider>
        ),
      })
      expect(result.current.themeName).toMatch('Light')
    })
    it('sets theme as dark', () => {
      const { result } = renderHook(() => useCurrentTheme(), {
        wrapper: ({ children }) => (
          <Provider store={createStore(themeReducer, { theme: 'DarkTheme' })}>
            {children}
          </Provider>
        ),
      })

      expect(result.current.themeName).toMatch('Dark')
    })
    it('sets theme as light', () => {
      const { result } = renderHook(() => useCurrentTheme(), {
        wrapper: ({ children }) => (
          <Provider store={createStore(themeReducer, { theme: 'LightTheme' })}>
            {children}
          </Provider>
        ),
      })

      expect(result.current.themeName).toMatch('Light')
    })
    it('sets theme as spotify-ish', () => {
      const { result } = renderHook(() => useCurrentTheme(), {
        wrapper: ({ children }) => (
          <Provider
            store={createStore(themeReducer, { theme: 'SpotifyTheme' })}
          >
            {children}
          </Provider>
        ),
      })

      expect(result.current.themeName).toMatch('Spotify-ish')
    })
  })
  describe('with user preference theme as dark', () => {
    beforeAll(() => {
      window.matchMedia = createMatchMedia('dark')
    })
    it('sets theme as dark in auto mode', () => {
      const { result } = renderHook(() => useCurrentTheme(), {
        wrapper: ({ children }) => (
          <Provider store={createStore(themeReducer, { theme: AUTO_THEME_ID })}>
            {children}
          </Provider>
        ),
      })

      expect(result.current.themeName).toMatch('Dark')
    })
    it('sets theme as dark', () => {
      const { result } = renderHook(() => useCurrentTheme(), {
        wrapper: ({ children }) => (
          <Provider store={createStore(themeReducer, { theme: 'DarkTheme' })}>
            {children}
          </Provider>
        ),
      })

      expect(result.current.themeName).toMatch('Dark')
    })
    it('sets theme as light', () => {
      const { result } = renderHook(() => useCurrentTheme(), {
        wrapper: ({ children }) => (
          <Provider store={createStore(themeReducer, { theme: 'LightTheme' })}>
            {children}
          </Provider>
        ),
      })

      expect(result.current.themeName).toMatch('Light')
    })
    it('sets theme as spotify-ish', () => {
      const { result } = renderHook(() => useCurrentTheme(), {
        wrapper: ({ children }) => (
          <Provider
            store={createStore(themeReducer, { theme: 'SpotifyTheme' })}
          >
            {children}
          </Provider>
        ),
      })

      expect(result.current.themeName).toMatch('Spotify-ish')
    })
  })
})
