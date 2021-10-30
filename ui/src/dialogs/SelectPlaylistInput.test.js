import * as React from 'react'
import { TestContext } from 'ra-test'
import { DataProviderContext } from 'react-admin'
import { cleanup, fireEvent, render, waitFor } from '@testing-library/react'
import { SelectPlaylistInput } from './SelectPlaylistInput'

describe('SelectPlaylistInput', () => {
  beforeAll(() => localStorage.setItem('userId', 'admin'))
  afterEach(cleanup)
  const onChangeHandler = jest.fn()

  it('should call the handler with the selections', async () => {
    const mockData = [
      { id: 'sample-id1', name: 'sample playlist 1', ownerId: 'admin' },
      { id: 'sample-id2', name: 'sample playlist 2', ownerId: 'admin' },
    ]
    const mockIndexedData = {
      'sample-id1': {
        id: 'sample-id1',
        name: 'sample playlist 1',
        ownerId: 'admin',
      },
      'sample-id2': {
        id: 'sample-id2',
        name: 'sample playlist 2',
        ownerId: 'admin',
      },
    }

    const mockDataProvider = {
      getList: jest
        .fn()
        .mockResolvedValue({ data: mockData, total: mockData.length }),
    }

    render(
      <DataProviderContext.Provider value={mockDataProvider}>
        <TestContext
          initialState={{
            addToPlaylistDialog: { open: true, duplicateSong: false },
            admin: {
              ui: { optimistic: false },
              resources: {
                playlist: {
                  data: mockIndexedData,
                  list: {
                    cachedRequests: {
                      '{"pagination":{"page":1,"perPage":-1},"sort":{"field":"name","order":"ASC"},"filter":{}}':
                        {
                          ids: ['sample-id1', 'sample-id2'],
                          total: 2,
                        },
                    },
                  },
                },
              },
            },
          }}
        >
          <SelectPlaylistInput onChange={onChangeHandler} />
        </TestContext>
      </DataProviderContext.Provider>
    )

    await waitFor(() => {
      expect(mockDataProvider.getList).toHaveBeenCalledWith('playlist', {
        filter: {},
        pagination: { page: 1, perPage: -1 },
        sort: { field: 'name', order: 'ASC' },
      })
    })

    fireEvent.change(document.activeElement, { target: { value: 'sample' } })
    fireEvent.keyDown(document.activeElement, { key: 'ArrowDown' })
    fireEvent.keyDown(document.activeElement, { key: 'Enter' })
    await waitFor(() => {
      expect(onChangeHandler).toHaveBeenCalledWith([
        { id: 'sample-id1', name: 'sample playlist 1', ownerId: 'admin' },
      ])
    })

    fireEvent.keyDown(document.activeElement, { key: 'ArrowDown' })
    fireEvent.keyDown(document.activeElement, { key: 'Enter' })
    await waitFor(() => {
      expect(onChangeHandler).toHaveBeenCalledWith([
        { id: 'sample-id1', name: 'sample playlist 1', ownerId: 'admin' },
        { id: 'sample-id2', name: 'sample playlist 2', ownerId: 'admin' },
      ])
    })

    fireEvent.change(document.activeElement, {
      target: { value: 'new playlist' },
    })
    fireEvent.keyDown(document.activeElement, { key: 'ArrowDown' })
    fireEvent.keyDown(document.activeElement, { key: 'Enter' })
    await waitFor(() => {
      expect(onChangeHandler).toHaveBeenCalledWith([
        { id: 'sample-id1', name: 'sample playlist 1', ownerId: 'admin' },
        { id: 'sample-id2', name: 'sample playlist 2', ownerId: 'admin' },
        { name: 'new playlist' },
      ])
    })

    fireEvent.change(document.activeElement, {
      target: { value: 'another new playlist' },
    })
    fireEvent.keyDown(document.activeElement, { key: 'Enter' })
    await waitFor(() => {
      expect(onChangeHandler).toHaveBeenCalledWith([
        { id: 'sample-id1', name: 'sample playlist 1', ownerId: 'admin' },
        { id: 'sample-id2', name: 'sample playlist 2', ownerId: 'admin' },
        { name: 'new playlist' },
        { name: 'another new playlist' },
      ])
    })
  })
})
