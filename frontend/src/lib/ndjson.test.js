import { describe, it, expect } from 'vitest'
import { splitLines } from './ndjson.js'

describe('splitLines — 单块若干情形', () => {
  it('单块含多行，最后一行带 \\n', () => {
    expect(splitLines('', 'a\nb\nc\n')).toEqual({
      lines: ['a', 'b', 'c'],
      buffer: '',
    })
  })

  it('单块末尾不带 \\n，最后一行进 buffer', () => {
    expect(splitLines('', 'a\nb\nc')).toEqual({
      lines: ['a', 'b'],
      buffer: 'c',
    })
  })

  it('空输入', () => {
    expect(splitLines('', '')).toEqual({ lines: [], buffer: '' })
  })

  it('空块 + 已有 buffer 维持原状', () => {
    expect(splitLines('half', '')).toEqual({ lines: [], buffer: 'half' })
  })

  it('只有 \\n 切出一个空行', () => {
    expect(splitLines('', '\n')).toEqual({ lines: [''], buffer: '' })
  })

  it('连续空行保留为多条空字符串', () => {
    expect(splitLines('', 'a\n\nb\n')).toEqual({
      lines: ['a', '', 'b'],
      buffer: '',
    })
  })
})

describe('splitLines — 跨块拼接', () => {
  it('一行被切成两块，第二块带 \\n 才算完整', () => {
    const r1 = splitLines('', '{"par')
    expect(r1).toEqual({ lines: [], buffer: '{"par' })

    const r2 = splitLines(r1.buffer, 'tial":1}\n')
    expect(r2).toEqual({ lines: ['{"partial":1}'], buffer: '' })
  })

  it('一行被切成三块', () => {
    const r1 = splitLines('', 'aaa')
    const r2 = splitLines(r1.buffer, 'bbb')
    const r3 = splitLines(r2.buffer, 'ccc\n')
    expect(r1).toEqual({ lines: [], buffer: 'aaa' })
    expect(r2).toEqual({ lines: [], buffer: 'aaabbb' })
    expect(r3).toEqual({ lines: ['aaabbbccc'], buffer: '' })
  })

  it('块尾正好落在 \\n 上', () => {
    const r1 = splitLines('', 'first\n')
    const r2 = splitLines(r1.buffer, 'second\n')
    expect(r1).toEqual({ lines: ['first'], buffer: '' })
    expect(r2).toEqual({ lines: ['second'], buffer: '' })
  })

  it('一块里既有完整行又有残行', () => {
    const r = splitLines('', 'whole\nstill-')
    expect(r).toEqual({ lines: ['whole'], buffer: 'still-' })
    const r2 = splitLines(r.buffer, 'going\nrest')
    expect(r2).toEqual({ lines: ['still-going'], buffer: 'rest' })
  })
})

describe('splitLines — CRLF', () => {
  it('CRLF 行尾被剥掉', () => {
    expect(splitLines('', 'a\r\nb\r\n')).toEqual({
      lines: ['a', 'b'],
      buffer: '',
    })
  })

  it('混合 LF 与 CRLF', () => {
    expect(splitLines('', 'a\nb\r\nc\n')).toEqual({
      lines: ['a', 'b', 'c'],
      buffer: '',
    })
  })

  it('CRLF 跨块拼接', () => {
    const r1 = splitLines('', 'a\r')
    expect(r1).toEqual({ lines: [], buffer: 'a\r' })
    const r2 = splitLines(r1.buffer, '\n')
    expect(r2).toEqual({ lines: ['a'], buffer: '' })
  })
})

describe('splitLines — JSON 行实战', () => {
  it('两个完整 JSON 行 + 末尾残行', () => {
    const r = splitLines('', '{"type":"summary","summary":"hi"}\n{"type":"risks","risks":[]}\n{"par')
    expect(r.lines).toEqual([
      '{"type":"summary","summary":"hi"}',
      '{"type":"risks","risks":[]}',
    ])
    expect(r.buffer).toBe('{"par')
  })
})
