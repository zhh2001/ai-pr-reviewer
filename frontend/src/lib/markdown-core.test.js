import { describe, it, expect } from 'vitest'
import { renderMarkdownUnsanitized } from './markdown-core.js'

describe('renderMarkdownUnsanitized', () => {
  it('escapes raw <script> blocks', () => {
    const out = renderMarkdownUnsanitized('<script>alert(1)</script>')
    expect(out).not.toContain('<script>')
    expect(out).toContain('&lt;script&gt;')
  })

  it('escapes <img onerror> attacks', () => {
    const out = renderMarkdownUnsanitized('<img src=x onerror=alert(1)>')
    expect(out).not.toMatch(/<img\b/i)
    expect(out).toContain('&lt;img')
  })

  it('does not produce javascript: hrefs', () => {
    // markdown-it 拒掉危险 scheme 后会把 [text](url) 当成普通文本——url 字符串本身
    // 还在输出里，但不会成为可点的 <a href=...>。这里只关心后者。
    const out = renderMarkdownUnsanitized('[click](javascript:alert(1))')
    expect(out).not.toMatch(/href\s*=\s*["']?javascript:/i)
  })

  it('does not produce data: hrefs', () => {
    const out = renderMarkdownUnsanitized('[x](data:text/html,<script>1</script>)')
    expect(out).not.toMatch(/href\s*=\s*["']?data:/i)
    // 内嵌的 <script> 仍要被实体化。
    expect(out).not.toContain('<script>')
    expect(out).toContain('&lt;script&gt;')
  })

  it('keeps http(s) hrefs', () => {
    const out = renderMarkdownUnsanitized('[ok](https://example.com/path)')
    expect(out).toMatch(/href\s*=\s*["']https:\/\/example\.com\/path["']/)
  })

  it('renders bold / italic / inline code', () => {
    expect(renderMarkdownUnsanitized('**bold**')).toContain('<strong>bold</strong>')
    expect(renderMarkdownUnsanitized('*italic*')).toContain('<em>italic</em>')
    expect(renderMarkdownUnsanitized('`x`')).toContain('<code>x</code>')
  })

  it('renders fenced code blocks', () => {
    const out = renderMarkdownUnsanitized('```js\nconst x = 1\n```')
    expect(out).toContain('<pre')
    expect(out).toContain('<code')
    expect(out).toContain('const x = 1')
  })

  it('renders bullet lists', () => {
    const out = renderMarkdownUnsanitized('- a\n- b')
    expect(out).toContain('<ul>')
    expect(out).toContain('<li>a</li>')
    expect(out).toContain('<li>b</li>')
  })

  it('handles nullish / empty input', () => {
    expect(renderMarkdownUnsanitized('')).toBe('')
    expect(renderMarkdownUnsanitized(null)).toBe('')
    expect(renderMarkdownUnsanitized(undefined)).toBe('')
  })

  it('keeps plain text safe even with HTML-looking content', () => {
    const out = renderMarkdownUnsanitized('a < b && c > d')
    expect(out).toContain('&lt;')
    expect(out).toContain('&gt;')
    expect(out).toContain('&amp;')
  })
})
