import React from 'react'
import { render, screen } from '@testing-library/react'
import { createMuiTheme, ThemeProvider } from '@material-ui/core/styles'
import { Pagination } from 'react-admin'
import { describe, it, expect } from 'vitest'
import themes from './index'

const VISUAL_PROPS =
  /^(color|background(-color)?|border(-(top|right|bottom|left))?(-(color|style|width))?|font-weight|text-decoration.*|box-shadow|outline.*|opacity)$/

const specificity = (selector) => [
  (selector.match(/#[\w-]+/g) || []).length,
  (selector.match(/\.[\w-]+|\[[^\]]*\]|:(?!:)[\w-]+/g) || []).length,
  (selector.match(/(^|[\s>+~])[a-z][\w-]*/gi) || []).length,
]

const compare = (a, b) => {
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return a[i] - b[i]
  return 0
}

const matchingSpecificity = (el, selectorText) => {
  const matches = selectorText
    .split(',')
    .map((s) => s.trim())
    .filter((s) => {
      try {
        return el.matches(s)
      } catch {
        return false
      }
    })
    .map(specificity)
    .sort(compare)
  return matches.pop()
}

// jsdom's getComputedStyle ignores most JSS rules, and document.styleSheets is in creation
// order, not DOM order, so resolve the cascade by hand from the <style> tags
const visualStyle = (el) => {
  const winners = {}
  let order = 0
  for (const style of Array.from(document.querySelectorAll('style'))) {
    for (const rule of Array.from(style.sheet?.cssRules ?? [])) {
      order++
      if (!rule.selectorText) continue
      const spec = matchingSpecificity(el, rule.selectorText)
      if (!spec) continue
      for (const prop of Array.from(rule.style)) {
        if (!VISUAL_PROPS.test(prop)) continue
        const important = rule.style.getPropertyPriority(prop) === 'important'
        const rank = [important ? 1 : 0, ...spec, order]
        if (!winners[prop] || compare(rank, winners[prop].rank) >= 0) {
          winners[prop] = { rank, value: rule.style.getPropertyValue(prop) }
        }
      }
    }
  }
  return Object.fromEntries(
    Object.entries(winners).map(([prop, { value }]) => [prop, value]),
  )
}

describe('Pagination current page', () => {
  it.each(Object.keys(themes))('stands out in %s', (themeName) => {
    render(
      <ThemeProvider theme={createMuiTheme(themes[themeName])}>
        <Pagination
          page={3}
          perPage={10}
          total={100}
          setPage={() => {}}
          setPerPage={() => {}}
        />
      </ThemeProvider>,
    )
    const look = (name) => {
      const button = screen.getByRole('button', { name })
      return [visualStyle(button), visualStyle(button.firstChild)]
    }
    expect(look('3')).not.toEqual(look('4'))
  })
})
