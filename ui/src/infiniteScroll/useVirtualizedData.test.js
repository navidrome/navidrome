import { expect, it, jest } from '@jest/globals'
import { renderHook } from '@testing-library/react-hooks'
import useVirtualizedData from './useVirtualizedData'
import { TestContext } from 'ra-test'
import { DataProviderContext, ListContextProvider } from 'ra-core'

const renderVirtualizedHook = (dataProvider = {}) => {
  const initialState = {
    admin: {
      resources: {
        users: {
          data: {
            1: { id: 1, name: 'user1' },
            2: { id: 2, name: 'user2' },
            3: { id: 3, name: 'user3' },
          },
          list: {
            ids: [1, 2, 3],
            total: 3,
          },
        },
      },
    },
  }
  const listProps = {
    basePath: '/',
    resource: 'users',
    location: {},
    match: {},
    perPage: 3,
  }

  return renderHook(() => useVirtualizedData(), {
    wrapper: ({ children }) => (
      <TestContext initialState={initialState}>
        <DataProviderContext.Provider value={dataProvider}>
          <ListContextProvider value={listProps}>
            {children}
          </ListContextProvider>
        </DataProviderContext.Provider>
      </TestContext>
    ),
  })
}
describe('useVirtualizedData', () => {
  it('does not request first page', () => {
    const dataProvider = {
      getList: jest.fn().mockResolvedValue({
        data: [
          {
            id: 1,
            name: 'user1',
          },
          {
            id: 2,
            name: 'user2',
          },
        ],
        total: 5,
      }),
    }

    const { result } = renderVirtualizedHook(dataProvider)
    const firstPageQuery = { startIndex: 0, stopIndex: 1 }

    expect(result.current.handleLoadMore(firstPageQuery)).toEqual(undefined)
    expect(dataProvider.getList).not.toBeCalled()
  })

  it('fetches and populates data correctly', async () => {
    const dataProvider = {
      getList: jest.fn().mockResolvedValue({
        data: [
          {
            id: 4,
            name: 'user4',
          },
          {
            id: 5,
            name: 'user5',
          },
        ],
        total: 5,
      }),
    }
    const { result, waitForNextUpdate } = renderVirtualizedHook(dataProvider)

    const dataQuery = { startIndex: 3, stopIndex: 4 }
    result.current.handleLoadMore(dataQuery)

    await waitForNextUpdate()

    expect(result.current.loadedIds).toEqual({
      0: 1,
      1: 2,
      2: 3,
      3: 4,
      4: 5,
    })

    expect(dataProvider.getList).toBeCalledTimes(1)
    expect(dataProvider.getList).toBeCalledWith(
      'users',
      expect.objectContaining({
        pagination: {
          page: 2,
          perPage: 3,
        },
      })
    )
  })
})
