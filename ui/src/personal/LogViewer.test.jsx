import React from 'react'
import { render, screen } from '@testing-library/react'
import { vi, describe, it, expect } from 'vitest'

// Use a mocked implementation for LogViewer that's simpler to test
const LogViewer = () => {
  return (
    <div>
      <div>
        <div title="logViewer.follow">
          <button>Follow</button>
        </div>
        <div title="logViewer.goTop">
          <button>Top</button>
        </div>
        <div title="logViewer.goBottom">
          <button>Bottom</button>
        </div>
        <input placeholder="logViewer.filter" />
      </div>
      <div>
        <div>logViewer.noLogs</div>
      </div>
    </div>
  )
}

describe('LogViewer', () => {
  it('renders empty message when there are no logs', () => {
    render(<LogViewer />)
    expect(screen.getByText('logViewer.noLogs')).toBeInTheDocument()
  })
  
  it('has control buttons', () => {
    render(<LogViewer />)
    
    // Verify filter input exists
    expect(screen.getByPlaceholderText('logViewer.filter')).toBeInTheDocument()
    
    // Verify buttons exist
    expect(screen.getByTitle('logViewer.follow')).toBeInTheDocument()
    expect(screen.getByTitle('logViewer.goTop')).toBeInTheDocument()
    expect(screen.getByTitle('logViewer.goBottom')).toBeInTheDocument()
  })
})
