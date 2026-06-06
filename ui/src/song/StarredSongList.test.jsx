import React from 'react'
import { render, screen } from '@testing-library/react'
import StarredSongList from './StarredSongList'

vi.mock('react-admin', () => ({
  ResourceContextProvider: ({ children }) => <div>{children}</div>,
}))

vi.mock('./SongList', () => ({
  default: (props) => (
    <div
      data-testid="song-list"
      data-resource={props.resource}
      data-filter={JSON.stringify(props.filter)}
      data-sort={JSON.stringify(props.sort)}
    />
  ),
}))

describe('<StarredSongList />', () => {
  it('renders the song list scoped to starred songs', () => {
    render(<StarredSongList />)

    const list = screen.getByTestId('song-list')
    expect(list.getAttribute('data-resource')).toBe('song')
    expect(JSON.parse(list.getAttribute('data-filter'))).toEqual({
      starred: true,
    })
    expect(JSON.parse(list.getAttribute('data-sort'))).toEqual({
      field: 'starred_at',
      order: 'DESC',
    })
  })
})
