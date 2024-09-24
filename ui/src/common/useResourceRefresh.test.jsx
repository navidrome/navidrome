import { vi } from 'vitest'
import * as React from 'react'
import * as Redux from 'react-redux'
import * as RA from 'react-admin'
import { useResourceRefresh } from './useResourceRefresh'

vi.mock('react', async () => {
  const actual = await vi.importActual('react')
  return {
    ...actual,
    useState: vi.fn(),
  }
})

vi.mock('react-redux', async () => {
  const actual = await vi.importActual('react-redux')
  return {
    ...actual,
    useSelector: vi.fn(),
  }
})

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useRefresh: vi.fn(),
    useDataProvider: vi.fn(),
  }
})

describe('useResourceRefresh', () => {
  const setState = vi.fn()
  const useStateMock = (initState) => [initState, setState]
  const refresh = vi.fn()
  const useRefreshMock = () => refresh
  const getMany = vi.fn()
  const useDataProviderMock = () => ({ getMany })
  let lastTime

  beforeEach(() => {
    vi.spyOn(React, 'useState').mockImplementation(useStateMock)
    vi.spyOn(RA, 'useRefresh').mockImplementation(useRefreshMock)
    vi.spyOn(RA, 'useDataProvider').mockImplementation(useDataProviderMock)
    lastTime = new Date(new Date().valueOf() + 1000)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('stores last time checked, to avoid redundant runs', () => {
    const useSelectorMock = () => ({ lastReceived: lastTime })
    vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

    useResourceRefresh()

    expect(setState).toHaveBeenCalledWith(lastTime)
  })

  it("does not run again if lastTime didn't change", () => {
    vi.spyOn(React, 'useState').mockImplementation(() => [lastTime, setState])
    const useSelectorMock = () => ({ lastReceived: lastTime })
    vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

    useResourceRefresh()

    expect(setState).not.toHaveBeenCalled()
  })

  describe('No visible resources specified', () => {
    it('triggers a UI refresh when received a "any" resource refresh', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { '*': '*' },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useResourceRefresh()

      expect(refresh).toHaveBeenCalledTimes(1)
      expect(getMany).not.toHaveBeenCalled()
    })

    it('triggers a UI refresh when received an "any" id', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { album: ['*'] },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useResourceRefresh()

      expect(refresh).toHaveBeenCalledTimes(1)
      expect(getMany).not.toHaveBeenCalled()
    })

    it('triggers a refetch of the resources received', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { album: ['al-1', 'al-2'], song: ['sg-1', 'sg-2'] },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useResourceRefresh()

      expect(refresh).not.toHaveBeenCalled()
      expect(getMany).toHaveBeenCalledTimes(2)
      expect(getMany).toHaveBeenCalledWith('album', { ids: ['al-1', 'al-2'] })
      expect(getMany).toHaveBeenCalledWith('song', { ids: ['sg-1', 'sg-2'] })
    })
  })

  describe('Visible resources specified', () => {
    it('triggers a UI refresh when received a "any" resource refresh', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { '*': '*' },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useResourceRefresh('album')

      expect(refresh).toHaveBeenCalledTimes(1)
      expect(getMany).not.toHaveBeenCalled()
    })

    it('triggers a refetch of the resources received if they are visible', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { album: ['al-1', 'al-2'], song: ['sg-1', 'sg-2'] },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useResourceRefresh('song')

      expect(refresh).not.toHaveBeenCalled()
      expect(getMany).toHaveBeenCalledTimes(1)
      expect(getMany).toHaveBeenCalledWith('song', { ids: ['sg-1', 'sg-2'] })
    })
  })
})
