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
