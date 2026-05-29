import { describe, it, expect } from 'vitest'
import { severityRank, severityClass, sortBySeverity } from './severity.js'

describe('severityRank', () => {
  it('orders high < medium < low', () => {
    expect(severityRank('high')).toBeLessThan(severityRank('medium'))
    expect(severityRank('medium')).toBeLessThan(severityRank('low'))
  })

  it('is case-insensitive', () => {
    expect(severityRank('HIGH')).toBe(severityRank('high'))
    expect(severityRank('Medium')).toBe(severityRank('medium'))
  })

  it('puts unknown / missing values last', () => {
    expect(severityRank('weird')).toBe(99)
    expect(severityRank(undefined)).toBe(99)
    expect(severityRank(null)).toBe(99)
    expect(severityRank('')).toBe(99)
  })
})

describe('severityClass', () => {
  it('maps known levels to sev-*', () => {
    expect(severityClass('high')).toBe('sev-high')
    expect(severityClass('MEDIUM')).toBe('sev-medium')
    expect(severityClass('low')).toBe('sev-low')
  })

  it('falls back to sev-unknown for anything else', () => {
    expect(severityClass('critical')).toBe('sev-unknown')
    expect(severityClass(undefined)).toBe('sev-unknown')
  })
})

describe('sortBySeverity', () => {
  it('sorts high → medium → low', () => {
    const sorted = sortBySeverity([
      { severity: 'low', id: 1 },
      { severity: 'high', id: 2 },
      { severity: 'medium', id: 3 },
    ])
    expect(sorted.map((r) => r.severity)).toEqual(['high', 'medium', 'low'])
  })

  it('places unknown severities after known ones', () => {
    const sorted = sortBySeverity([
      { severity: '???' },
      { severity: 'low' },
      { severity: 'high' },
    ])
    expect(sorted.map((r) => r.severity)).toEqual(['high', 'low', '???'])
  })

  it('does not mutate input', () => {
    const input = [{ severity: 'low' }, { severity: 'high' }]
    sortBySeverity(input)
    expect(input.map((r) => r.severity)).toEqual(['low', 'high'])
  })

  it('handles nullish input', () => {
    expect(sortBySeverity(null)).toEqual([])
    expect(sortBySeverity(undefined)).toEqual([])
  })
})
