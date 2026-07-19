import { describe, it, expect, beforeEach } from 'vitest'
import { getStoredPerPage, setStoredPerPage } from './perPageStore'

const options = [15, 25, 50]

describe('perPageStore', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('round-trips a stored value', () => {
    setStoredPerPage('song', 25)
    expect(getStoredPerPage('song', options, 15)).toEqual(25)
  })

  it('keys values per resource', () => {
    setStoredPerPage('song', 25)
    setStoredPerPage('playlist', 50)
    expect(getStoredPerPage('song', options, 15)).toEqual(25)
    expect(getStoredPerPage('playlist', options, 15)).toEqual(50)
  })

  it('returns the fallback when nothing is stored', () => {
    expect(getStoredPerPage('song', options, 15)).toEqual(15)
  })

  it('returns the fallback for garbage values', () => {
    localStorage.setItem('perPage.song', 'bogus')
    expect(getStoredPerPage('song', options, 15)).toEqual(15)
  })

  it('returns the fallback when the stored value is not a valid option', () => {
    setStoredPerPage('album', 90)
    expect(getStoredPerPage('album', [18, 36, 72], 18)).toEqual(18)
  })
})
