import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'
import Linkify from './Linkify'

const URL = 'http://www.example.com'

const expectLink = (url) => {
  const linkEl = screen.getByRole('link')
  expect(linkEl).not.toBeNull()
  expect(linkEl?.href).toBe(url)
}

describe('<Linkify />', () => {
  it('should render link', () => {
    render(<Linkify text={URL} />)
    expectLink(`${URL}/`)
    expect(screen.getByText(URL)).toBeInTheDocument()
  })

  it('should render link and text', () => {
    render(<Linkify text={`foo ${URL} bar`} />)
    expectLink(`${URL}/`)
    expect(screen.getByText(/foo/i)).toBeInTheDocument()
    expect(screen.getByText(URL)).toBeInTheDocument()
    expect(screen.getByText(/bar/i)).toBeInTheDocument()
  })

  it('should render only text', () => {
    render(<Linkify text={'foo bar'} />)
    expect(screen.queryAllByRole('link')).toHaveLength(0)
    expect(screen.getByText(/foo bar/i)).toBeInTheDocument()
  })
})
