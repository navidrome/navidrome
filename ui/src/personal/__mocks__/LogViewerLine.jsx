import React from 'react'

const LogViewerLine = ({ index, style, data }) => {
  const log = data.logs[index]
  return (
    <div data-testid={`log-line-${index}`}>
      <span className="level">{log.level}</span>
      <span className="message">{log.message}</span>
    </div>
  )
}

export default LogViewerLine