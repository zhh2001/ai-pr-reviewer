import { describe, it, expect } from 'vitest'
import { confidenceBarPercent, groupByCategory } from './format.js'

describe('confidenceBarPercent', () => {
  it('clamps to [0,100]', () => {
    expect(confidenceBarPercent(0)).toBe(0)
    expect(confidenceBarPercent(1)).toBe(100)
    expect(confidenceBarPercent(0.5)).toBe(50)
    expect(confidenceBarPercent(-0.3)).toBe(0)
    expect(confidenceBarPercent(2)).toBe(100)
  })
  it('rounds to integer', () => {
    expect(confidenceBarPercent(0.756)).toBe(76)
    expect(confidenceBarPercent(0.123)).toBe(12)
  })
  it('handles null / undefined / NaN', () => {
    expect(confidenceBarPercent(null)).toBe(0)
    expect(confidenceBarPercent(undefined)).toBe(0)
    expect(confidenceBarPercent(NaN)).toBe(0)
    expect(confidenceBarPercent('not a number')).toBe(0)
  })
})

describe('groupByCategory', () => {
  it('groups in first-seen order', () => {
    const out = groupByCategory([
      { category: 'testing', t: 'a' },
      { category: 'naming', t: 'b' },
      { category: 'testing', t: 'c' },
      { category: 'naming', t: 'd' },
    ])
    expect(out.map((g) => g.category)).toEqual(['testing', 'naming'])
    expect(out[0].items.map((i) => i.t)).toEqual(['a', 'c'])
    expect(out[1].items.map((i) => i.t)).toEqual(['b', 'd'])
  })
  it('maps missing category to "other"', () => {
    const out = groupByCategory([{ t: 'x' }, { category: '', t: 'y' }])
    expect(out).toEqual([{ category: 'other', items: [{ t: 'x' }, { category: '', t: 'y' }] }])
  })
  it('handles empty / nullish input', () => {
    expect(groupByCategory([])).toEqual([])
    expect(groupByCategory(null)).toEqual([])
    expect(groupByCategory(undefined)).toEqual([])
  })
})
