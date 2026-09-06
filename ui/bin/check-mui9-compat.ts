import { readdir, readFile } from 'node:fs/promises'
import { extname, join, relative, resolve } from 'node:path'
import ts from '@typescript/typescript6'

const sourceRoot = resolve(import.meta.dir, '../src')
const sourceExtensions = new Set(['.js', '.jsx', '.ts', '.tsx'])

const legacyJavaScript = (
  await readdir(sourceRoot, { recursive: true })
).filter((file) => file.endsWith('.js') || file.endsWith('.jsx'))
if (legacyJavaScript.length) {
  for (const file of legacyJavaScript) {
    process.stderr.write(
      `${file}: JavaScript source is forbidden; use TypeScript\n`,
    )
  }
  process.exit(1)
}

const removedProps = new Map<string, Set<string>>([
  ['Accordion', new Set(['TransitionComponent', 'TransitionProps'])],
  ['Alert', new Set(['components', 'componentsProps'])],
  [
    'Autocomplete',
    new Set([
      'ChipProps',
      'componentsProps',
      'ListboxComponent',
      'ListboxProps',
      'PaperComponent',
      'PopperComponent',
      'renderTags',
    ]),
  ],
  ['Avatar', new Set(['imgProps'])],
  ['AvatarGroup', new Set(['componentsProps'])],
  [
    'Backdrop',
    new Set(['components', 'componentsProps', 'TransitionComponent']),
  ],
  ['Badge', new Set(['components', 'componentsProps'])],
  ['Dialog', new Set(['disableEscapeKeyDown'])],
  ['Divider', new Set(['light'])],
  ['FormControlLabel', new Set(['componentsProps'])],
  ['ListItem', new Set(['button', 'components', 'componentsProps'])],
  ['Menu', new Set(['MenuListProps', 'PaperProps', 'TransitionProps'])],
  [
    'Modal',
    new Set([
      'BackdropComponent',
      'BackdropProps',
      'components',
      'componentsProps',
      'disableEscapeKeyDown',
    ]),
  ],
  ['MobileStepper', new Set(['LinearProgressProps'])],
  [
    'Popover',
    new Set([
      'BackdropComponent',
      'BackdropProps',
      'PaperProps',
      'TransitionComponent',
      'TransitionProps',
    ]),
  ],
  ['Popper', new Set(['components', 'componentsProps'])],
  ['Slider', new Set(['components', 'componentsProps'])],
  ['SpeedDial', new Set(['TransitionComponent', 'TransitionProps'])],
  [
    'SpeedDialAction',
    new Set([
      'FabProps',
      'tooltipTitle',
      'tooltipPlacement',
      'tooltipOpen',
      'TooltipClasses',
    ]),
  ],
  [
    'Tabs',
    new Set([
      'ScrollButtonComponent',
      'TabIndicatorProps',
      'TabScrollButtonProps',
    ]),
  ],
  [
    'TextField',
    new Set([
      'FormHelperTextProps',
      'InputLabelProps',
      'InputProps',
      'SelectProps',
      'inputProps',
    ]),
  ],
  [
    'Tooltip',
    new Set([
      'PopperComponent',
      'PopperProps',
      'TransitionComponent',
      'TransitionProps',
      'components',
      'componentsProps',
    ]),
  ],
  ['Typography', new Set(['paragraph'])],
])

const removedGridProps = new Set(['item', 'xs', 'sm', 'md', 'lg', 'xl'])
const removedSystemProps = new Set([
  'm',
  'mt',
  'mr',
  'mb',
  'ml',
  'mx',
  'my',
  'p',
  'pt',
  'pr',
  'pb',
  'pl',
  'px',
  'py',
  'display',
  'alignItems',
  'alignContent',
  'justifyItems',
  'justifyContent',
  'flex',
  'flexBasis',
  'flexDirection',
  'flexGrow',
  'flexShrink',
  'flexWrap',
  'gap',
  'rowGap',
  'columnGap',
  'fontFamily',
  'fontSize',
  'fontStyle',
  'fontWeight',
  'textAlign',
  'width',
  'minWidth',
  'maxWidth',
  'height',
  'minHeight',
  'maxHeight',
  'position',
  'top',
  'right',
  'bottom',
  'left',
  'zIndex',
])
const systemPropComponents = new Set([
  'Box',
  'DialogContentText',
  'Grid',
  'Link',
  'Stack',
  'Typography',
])

const removedIconNames = new Set([
  'AddCircleOutline',
  'ChatBubbleOutline',
  'CheckCircleOutline',
  'DeleteOutline',
  'DoneOutline',
  'DriveFileMoveOutline',
  'ErrorOutline',
  'HelpOutline',
  'InfoOutline',
  'LabelImportantOutline',
  'LightbulbOutline',
  'LockOutline',
  'MailOutline',
  'ModeEditOutline',
  'PauseCircleOutline',
  'PeopleOutline',
  'PersonOutline',
  'PieChartOutline',
  'PlayCircleOutline',
  'RemoveCircleOutline',
  'StarOutline',
  'WorkOutline',
  'WorkspacesOutline',
])

type Finding = { file: string; line: number; message: string }

const listSources = async (directory: string): Promise<string[]> => {
  const entries = await readdir(directory, { withFileTypes: true })
  const files = await Promise.all(
    entries.map((entry) => {
      const path = join(directory, entry.name)
      return entry.isDirectory() ? listSources(path) : [path]
    }),
  )
  return files.flat().filter((file) => sourceExtensions.has(extname(file)))
}

const getAttributeName = (attribute: ts.JsxAttributeLike) =>
  ts.isJsxAttribute(attribute) ? attribute.name.getText() : null

const isZero = (node: ts.Node | undefined) =>
  Boolean(node && ts.isNumericLiteral(node) && Number(node.text) === 0)

const findings: Finding[] = []
for (const file of await listSources(sourceRoot)) {
  const text = await readFile(file, 'utf8')
  const sourceFile = ts.createSourceFile(
    file,
    text,
    ts.ScriptTarget.Latest,
    true,
    file.endsWith('x') ? ts.ScriptKind.TSX : ts.ScriptKind.TS,
  )
  const muiComponents = new Map<string, string>()

  for (const statement of sourceFile.statements) {
    if (!ts.isImportDeclaration(statement)) continue
    const moduleName = statement.moduleSpecifier.getText().slice(1, -1)
    const clause = statement.importClause
    if (!clause) continue

    if (moduleName === '@mui/material' || moduleName === '@mui/lab') {
      const bindings = clause.namedBindings
      if (bindings && ts.isNamedImports(bindings)) {
        for (const element of bindings.elements) {
          muiComponents.set(
            element.name.text,
            element.propertyName?.text ?? element.name.text,
          )
        }
      }
    } else if (moduleName === '@mui/material/styles') {
      const bindings = clause.namedBindings
      if (bindings && ts.isNamedImports(bindings)) {
        for (const element of bindings.elements) {
          if (
            (element.propertyName?.text ?? element.name.text) === 'adaptV4Theme'
          ) {
            const { line } = sourceFile.getLineAndCharacterOfPosition(
              element.getStart(sourceFile),
            )
            findings.push({
              file: relative(sourceRoot, file),
              line: line + 1,
              message:
                'MUI 9 deprecated adaptV4Theme; modernize the theme shape',
            })
          }
        }
      }
    } else if (moduleName.startsWith('@mui/material/') && clause.name) {
      muiComponents.set(clause.name.text, moduleName.split('/').at(-1) ?? '')
    }

    if (moduleName.startsWith('@mui/icons-material/')) {
      const iconName = moduleName.split('/').at(-1) ?? ''
      if (removedIconNames.has(iconName)) {
        const { line } = sourceFile.getLineAndCharacterOfPosition(
          statement.getStart(sourceFile),
        )
        findings.push({
          file: relative(sourceRoot, file),
          line: line + 1,
          message: `MUI 9 removed icon export ${iconName}; use the Outlined variant`,
        })
      }
    }

    if (moduleName === '@mui/styles' || moduleName.startsWith('@mui/styles/')) {
      const { line } = sourceFile.getLineAndCharacterOfPosition(
        statement.getStart(sourceFile),
      )
      findings.push({
        file: relative(sourceRoot, file),
        line: line + 1,
        message: 'MUI 9 migration forbids @mui/styles; use styled() or sx',
      })
    }
  }

  const report = (node: ts.Node, message: string) => {
    const { line } = sourceFile.getLineAndCharacterOfPosition(
      node.getStart(sourceFile),
    )
    findings.push({ file: relative(sourceRoot, file), line: line + 1, message })
  }

  const visit = (node: ts.Node) => {
    if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
      const localName = node.tagName.getText(sourceFile)
      const component = muiComponents.get(localName)
      const attributes = node.attributes.properties

      for (const attribute of attributes) {
        const name = getAttributeName(attribute)
        if (!name) continue
        if (name === 'perPage' && ts.isJsxAttribute(attribute)) {
          const value = attribute.initializer
          if (value && ts.isJsxExpression(value) && isZero(value.expression)) {
            report(attribute, 'React-admin 5 perPage must not be zero')
          }
        }
        if (!component) continue
        if (removedProps.get(component)?.has(name)) {
          report(attribute, `MUI 9 removed ${component}.${name}`)
        }
        if (component === 'Grid' && removedGridProps.has(name)) {
          report(attribute, `MUI 9 Grid uses size instead of ${name}`)
        }
        if (
          systemPropComponents.has(component) &&
          removedSystemProps.has(name)
        ) {
          report(
            attribute,
            `MUI 9 removed ${component} system prop ${name}; use sx`,
          )
        }
      }
    }

    if (ts.isPropertyAssignment(node) && node.name.getText() === 'perPage') {
      if (isZero(node.initializer)) {
        report(node, 'React-admin 5 perPage must not be zero')
      }
    }

    if (
      ts.isCallExpression(node) &&
      node.expression.getText(sourceFile) === 'notify' &&
      node.arguments[1] &&
      ts.isStringLiteralLike(node.arguments[1])
    ) {
      report(
        node.arguments[1],
        'React-admin 5 notify options must be an object, not a type string',
      )
    }

    ts.forEachChild(node, visit)
  }
  visit(sourceFile)
}

if (findings.length) {
  for (const finding of findings) {
    process.stderr.write(
      `${finding.file}:${finding.line}: ${finding.message}\n`,
    )
  }
  process.exit(1)
}

process.stdout.write('MUI 9 and React-admin 5 compatibility audit passed\n')
