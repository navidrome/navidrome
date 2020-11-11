import React, { memo } from 'react'
import get from 'lodash.get'
import Typography from '@material-ui/core/Typography'
import sanitizeFieldRestProps from './sanitizeFieldRestProps'
import md5 from 'md5-hex'

const MultiLineTextField = memo(
  ({ className, emptyText, source, record = {}, stripTags, ...rest }) => {
    const value = get(record, source)
    const lines = value ? value.split('\n') : []

    return (
      <Typography
        className={className}
        variant="body2"
        component="span"
        {...sanitizeFieldRestProps(rest)}
      >
        {lines.length === 0 && emptyText
          ? emptyText
          : lines.map((line, idx) => (
              <div
                data-testid={`${source}.${idx}`}
                key={md5(line)}
                dangerouslySetInnerHTML={{ __html: line }}
              />
            ))}
      </Typography>
    )
  }
)

MultiLineTextField.defaultProps = {
  addLabel: true,
}

export default MultiLineTextField
