import { describe, it, expect } from 'vitest'
import { prFilesUrl } from './github.js'

describe('prFilesUrl', () => {
  it('builds the canonical Files-changed URL', () => {
    expect(prFilesUrl('owner', 'repo', 42)).toBe(
      'https://github.com/owner/repo/pull/42/files',
    )
  })
  it('encodes owner/repo with special chars', () => {
    expect(prFilesUrl('go org', 'repo.x', 7)).toBe(
      'https://github.com/go%20org/repo.x/pull/7/files',
    )
  })
  it('returns empty for missing pieces', () => {
    expect(prFilesUrl('', 'repo', 42)).toBe('')
    expect(prFilesUrl('owner', '', 42)).toBe('')
    expect(prFilesUrl('owner', 'repo', 0)).toBe('')
    expect(prFilesUrl('owner', 'repo', -1)).toBe('')
    expect(prFilesUrl(null, null, null)).toBe('')
  })
  it('does not embed a line anchor (intentionally)', () => {
    const url = prFilesUrl('o', 'r', 1)
    expect(url).not.toMatch(/#L\d+/)
    expect(url).not.toMatch(/#diff-/)
  })
})
