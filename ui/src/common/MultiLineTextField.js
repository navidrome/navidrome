import React, { memo } from 'react'
import Typography from '@material-ui/core/Typography'
import sanitizeFieldRestProps from './sanitizeFieldRestProps'
import md5 from 'md5-hex'

export const MultiLineTextField = memo(
  ({
    className,
    emptyText,
    source,
    record,
    firstLine,
    maxLines,
    addLabel,
    ...rest
  }) => {
    const value = record && record[source]
    let lines = value ? value.split('\n') : []
    if (maxLines || firstLine) {
      lines = lines.slice(firstLine, maxLines)
    }

    return (
      <Typography
        className={className}
        variant="body2"
        component="span"
        {...sanitizeFieldRestProps(rest)}
      >
        {lines.length === 0 && emptyText
          ? emptyText
          : lines.map((line, idx) =>
              line === '' ? (
                <br key={md5(line + idx)} />
              ) : (
                <div
                  data-testid={`${source}.${idx}`}
                  key={md5(line + idx)}
                  dangerouslySetInnerHTML={{ __html: line }}
                />
              )
            )}
      </Typography>
    )
  }
)

MultiLineTextField.defaultProps = {
  record: {},
  addLabel: true,
  firstLine: 0,
}
