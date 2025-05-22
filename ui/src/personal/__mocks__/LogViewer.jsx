import React from 'react'

const LogViewer = () => {
  return (
    <div data-testid="log-viewer">
      <div data-testid="filter-bar">
        <button aria-label="follow">Follow</button>
        <button aria-label="top">Top</button>
        <button aria-label="bottom">Bottom</button>
        <input placeholder="filter" />
      </div>
      <div data-testid="log-container">
        <div data-testid="no-logs">no logs</div>
      </div>
    </div>
  )
}

export default LogViewer
