// confidenceBarPercent: 把 [0,1] 的 confidence 映射到 [0,100] 的进度条宽度。
// 越界夹紧、非法回 0，避免给 CSS 喂出 NaN。
export function confidenceBarPercent(c) {
  if (c == null || Number.isNaN(Number(c))) return 0
  const n = Number(c)
  if (n <= 0) return 0
  if (n >= 1) return 100
  return Math.round(n * 100)
}

// groupByCategory: 把带 category 字段的对象按 category 分组，保留首次出现顺序。
// 缺失 category 归到 "other"。
export function groupByCategory(items) {
  if (!items || !items.length) return []
  const order = []
  const map = new Map()
  for (const it of items) {
    const cat = (it && it.category) || 'other'
    if (!map.has(cat)) {
      map.set(cat, [])
      order.push(cat)
    }
    map.get(cat).push(it)
  }
  return order.map((cat) => ({ category: cat, items: map.get(cat) }))
}
