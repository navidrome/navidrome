import React from 'react'
import { render, screen } from '@testing-library/react'
import { createMuiTheme, ThemeProvider } from '@material-ui/core/styles'
import { Pagination } from 'react-admin'
import { describe, it, expect } from 'vitest'
import themes from './index'

const VISUAL_PROPS =
  /^(color|background|border(?!-radius)|font-weight|text-decoration|box-shadow|outline|opacity)/

const specificity = (selector) =>
  [
    /#[\w-]+/g,
    /\.[\w-]+|\[[^\]]*\]|:(?!:)[\w-]+/g,
    /(^|[\s>+~])[a-z][\w-]*/gi,
  ].map((re) => (selector.match(re) || []).length)

const outranksOrTies = (a, b) => {
  const i = a.findIndex((v, k) => v !== b[k])
  return i === -1 || a[i] > b[i]
}

// jsdom's getComputedStyle ignores most JSS rules, and document.styleSheets is in creation
// order, not DOM order, so resolve the cascade by hand from the <style> tags
const visualStyle = (el) => {
  const style = {}
  const rank = {}
  for (const tag of Array.from(document.querySelectorAll('style'))) {
    for (const rule of Array.from(tag.sheet?.cssRules ?? [])) {
      let matches = false
      try {
        matches = !!rule.selectorText && el.matches(rule.selectorText)
      } catch {
        // jsdom rejects vendor pseudo-elements such as ::-moz-focus-inner
      }
      if (!matches) continue
      const spec = specificity(rule.selectorText)
      for (const prop of Array.from(rule.style)) {
        if (!VISUAL_PROPS.test(prop)) continue
        if (rank[prop] && !outranksOrTies(spec, rank[prop])) continue
        style[prop] = rule.style.getPropertyValue(prop)
        rank[prop] = spec
      }
    }
  }
  return style
}

describe('Pagination current page', () => {
  it.each(Object.entries(themes))('stands out in %s', (themeName, theme) => {
    render(
      <ThemeProvider theme={createMuiTheme(theme)}>
        <Pagination
          page={3}
          perPage={10}
          total={100}
          setPage={() => {}}
          setPerPage={() => {}}
        />
      </ThemeProvider>,
    )
    const current = visualStyle(screen.getByRole('button', { name: '3' }))
    const other = visualStyle(screen.getByRole('button', { name: '4' }))
    expect(current).not.toEqual(other)
  })
})
