// Severity 排序权重：high 最先出现。未知值排到最后。
const SEVERITY_ORDER = { high: 0, medium: 1, low: 2 }

function normalize(s) {
  return typeof s === 'string' ? s.toLowerCase() : ''
}

export function severityRank(severity) {
  const key = normalize(severity)
  return key in SEVERITY_ORDER ? SEVERITY_ORDER[key] : 99
}

export function severityClass(severity) {
  const key = normalize(severity)
  switch (key) {
    case 'high': return 'sev-high'
    case 'medium': return 'sev-medium'
    case 'low': return 'sev-low'
    default: return 'sev-unknown'
  }
}

// 返回新数组，不就地修改入参。
export function sortBySeverity(risks) {
  return [...(risks ?? [])].sort(
    (a, b) => severityRank(a?.severity) - severityRank(b?.severity),
  )
}

// severityCounts: 统计每个 severity 的条数；用于结果区顶部那条分布微条。
export function severityCounts(risks) {
  const out = { high: 0, medium: 0, low: 0, unknown: 0 }
  if (!risks || !risks.length) return out
  for (const r of risks) {
    const key = normalize(r?.severity)
    if (key === 'high' || key === 'medium' || key === 'low') {
      out[key]++
    } else {
      out.unknown++
    }
  }
  return out
}
