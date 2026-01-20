import React, { useCallback, useMemo } from 'react'
import {
  composePaths,
  computeLabel,
  createDefaultValue,
  isObjectArrayWithNesting,
  isPrimitiveArrayControl,
  rankWith,
  findUISchema,
  Resolve,
} from '@jsonforms/core'
import {
  JsonFormsDispatch,
  withJsonFormsArrayLayoutProps,
} from '@jsonforms/react'
import range from 'lodash/range'
import merge from 'lodash/merge'
import { Box, IconButton, Tooltip, Typography } from '@material-ui/core'
import { Add, Delete } from '@material-ui/icons'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles((theme) => ({
  arrayItem: {
    position: 'relative',
    padding: theme.spacing(2),
    marginBottom: theme.spacing(2),
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: theme.shape.borderRadius,
    '&:last-child': {
      marginBottom: 0,
    },
  },
  deleteButton: {
    position: 'absolute',
    top: theme.spacing(1),
    right: theme.spacing(1),
  },
  itemContent: {
    paddingRight: theme.spacing(4), // Space for delete button
  },
}))

// Default translations for array controls
const defaultTranslations = {
  addTooltip: 'Add',
  addAriaLabel: 'Add button',
  removeTooltip: 'Delete',
  removeAriaLabel: 'Delete button',
  noDataMessage: 'No data',
}

// Simplified array item renderer - clean card layout
// eslint-disable-next-line react-refresh/only-export-components
const ArrayItem = ({
  index,
  path,
  schema,
  uischema,
  uischemas,
  rootSchema,
  renderers,
  cells,
  enabled,
  removeItems,
  translations,
  disableRemove,
}) => {
  const classes = useStyles()
  const childPath = composePaths(path, `${index}`)

  const foundUISchema = useMemo(
    () =>
      findUISchema(
        uischemas,
        schema,
        uischema.scope,
        path,
        undefined,
        uischema,
        rootSchema,
      ),
    [uischemas, schema, path, uischema, rootSchema],
  )

  return (
    <Box className={classes.arrayItem}>
      {enabled && !disableRemove && (
        <Tooltip
          title={translations.removeTooltip}
          className={classes.deleteButton}
        >
          <IconButton
            onClick={() => removeItems(path, [index])()}
            size="small"
            aria-label={translations.removeAriaLabel}
          >
            <Delete fontSize="small" />
          </IconButton>
        </Tooltip>
      )}
      <Box className={classes.itemContent}>
        <JsonFormsDispatch
          enabled={enabled}
          schema={schema}
          uischema={foundUISchema}
          path={childPath}
          key={childPath}
          renderers={renderers}
          cells={cells}
        />
      </Box>
    </Box>
  )
}

// Array toolbar with add button
// eslint-disable-next-line react-refresh/only-export-components
const ArrayToolbar = ({
  label,
  description,
  enabled,
  addItem,
  path,
  createDefault,
  translations,
  disableAdd,
}) => (
  <Box mb={1}>
    <Box display="flex" alignItems="center" justifyContent="space-between">
      <Typography variant="h6">{label}</Typography>
      {!disableAdd && (
        <Tooltip
          title={translations.addTooltip}
          aria-label={translations.addAriaLabel}
        >
          <IconButton
            onClick={addItem(path, createDefault())}
            disabled={!enabled}
            size="small"
          >
            <Add />
          </IconButton>
        </Tooltip>
      )}
    </Box>
    {description && (
      <Typography variant="caption" color="textSecondary">
        {description}
      </Typography>
    )}
  </Box>
)

const useArrayStyles = makeStyles((theme) => ({
  container: {
    marginBottom: theme.spacing(2),
  },
}))

// Main array layout component - items always expanded
// eslint-disable-next-line react-refresh/only-export-components
const AlwaysExpandedArrayLayoutComponent = (props) => {
  const arrayClasses = useArrayStyles()
  const {
    enabled,
    data,
    path,
    schema,
    uischema,
    addItem,
    removeItems,
    renderers,
    cells,
    label,
    description,
    required,
    rootSchema,
    config,
    uischemas,
    disableAdd,
    disableRemove,
  } = props

  const innerCreateDefaultValue = useCallback(
    () => createDefaultValue(schema, rootSchema),
    [schema, rootSchema],
  )

  const appliedUiSchemaOptions = merge({}, config, uischema.options)
  const doDisableAdd = disableAdd || appliedUiSchemaOptions.disableAdd
  const doDisableRemove = disableRemove || appliedUiSchemaOptions.disableRemove
  const translations = defaultTranslations

  return (
    <div className={arrayClasses.container}>
      <ArrayToolbar
        translations={translations}
        label={computeLabel(
          label,
          required,
          appliedUiSchemaOptions.hideRequiredAsterisk,
        )}
        description={description}
        path={path}
        enabled={enabled}
        addItem={addItem}
        createDefault={innerCreateDefaultValue}
        disableAdd={doDisableAdd}
      />
      <div>
        {data > 0 ? (
          range(data).map((index) => (
            <ArrayItem
              key={index}
              index={index}
              path={path}
              schema={schema}
              uischema={uischema}
              uischemas={uischemas}
              rootSchema={rootSchema}
              renderers={renderers}
              cells={cells}
              enabled={enabled}
              removeItems={removeItems}
              translations={translations}
              disableRemove={doDisableRemove}
            />
          ))
        ) : (
          <Typography color="textSecondary">
            {translations.noDataMessage}
          </Typography>
        )}
      </div>
    </div>
  )
}

// Wrap with JSONForms HOC
const WrappedArrayLayout = withJsonFormsArrayLayoutProps(
  AlwaysExpandedArrayLayoutComponent,
)

// Custom tester that matches arrays but NOT enum arrays
// Enum arrays should be handled by MaterialEnumArrayRenderer (for checkboxes)
const isNonEnumArrayControl = (uischema, schema) => {
  // First check if it matches our base conditions (object array or primitive array)
  const baseCheck =
    isObjectArrayWithNesting(uischema, schema) ||
    isPrimitiveArrayControl(uischema, schema)

  if (!baseCheck) {
    return false
  }

  // Resolve the actual schema for this control using JSONForms utility
  const rootSchema = schema
  const resolved = Resolve.schema(rootSchema, uischema?.scope, rootSchema)

  // Exclude enum arrays (uniqueItems + oneOf/enum) - let MaterialEnumArrayRenderer handle them
  if (resolved?.uniqueItems && resolved?.items) {
    const { items } = resolved
    if (items.oneOf?.every((e) => e.const !== undefined) || items.enum) {
      return false
    }
  }

  return true
}

// Export as a renderer entry with high priority (5 > default 4)
// Matches both object arrays with nesting and primitive arrays, but NOT enum arrays
export const AlwaysExpandedArrayLayout = {
  tester: rankWith(5, isNonEnumArrayControl),
  renderer: WrappedArrayLayout,
}
