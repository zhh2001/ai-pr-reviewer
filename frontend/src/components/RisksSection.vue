<script setup>
import { computed } from 'vue'
import { sortBySeverity, severityClass } from '../severity.js'
import { confidenceBarPercent } from '../lib/format.js'
import { prFilesUrl } from '../lib/github.js'

const props = defineProps({
  risks: { type: Array, default: () => [] },
  risksError: { type: String, default: '' },
  risksFiltered: { type: Number, default: 0 },
  // 用于把 file:line chip 链到 GitHub 的 Files changed 页
  changes: { type: Object, default: () => ({}) },
})

const sorted = computed(() => sortBySeverity(props.risks ?? []))
const filesHref = computed(() =>
  prFilesUrl(props.changes?.owner, props.changes?.repo, props.changes?.number),
)

function formatConfidence(c) {
  if (c == null || Number.isNaN(Number(c))) return ''
  return `${Math.round(Number(c) * 100)}%`
}
</script>

<template>
  <section class="block">
    <h2>Risks</h2>

    <p v-if="risksError" class="section-error">
      本节分析暂时失败：{{ risksError }}
    </p>

    <template v-else>
      <p v-if="risksFiltered > 0" class="filtered-note">
        已过滤 {{ risksFiltered }} 条低置信度项（提高阈值才会显示更少，反之更多）
      </p>

      <ul v-if="sorted.length" class="risk-list">
        <li
          v-for="(r, i) in sorted"
          :key="i"
          class="risk-row"
          :class="severityClass(r.severity)"
        >
          <div class="head">
            <span class="pill" :class="severityClass(r.severity)">
              {{ (r.severity || '?').toUpperCase() }}
            </span>
            <a
              v-if="filesHref && r.file"
              class="loc-chip"
              :href="filesHref"
              target="_blank"
              rel="noopener noreferrer"
              :title="`Open ${r.file} on GitHub`"
            >
              {{ r.file }}<span v-if="r.line" class="line">:{{ r.line }}</span>
            </a>
            <span v-else class="loc-chip">
              {{ r.file || '?' }}<span v-if="r.line" class="line">:{{ r.line }}</span>
            </span>
            <span class="cat-tag">{{ r.category || '—' }}</span>
            <div class="conf" :title="`Confidence: ${formatConfidence(r.confidence)}`">
              <div class="conf-track">
                <div
                  class="conf-fill"
                  :class="severityClass(r.severity)"
                  :style="{ width: confidenceBarPercent(r.confidence) + '%' }"
                ></div>
              </div>
              <span class="conf-text mono">{{ formatConfidence(r.confidence) }}</span>
            </div>
          </div>
          <p class="desc">{{ r.description || '(no description)' }}</p>
        </li>
      </ul>
      <p v-else class="empty">No risks identified at the current threshold.</p>
    </template>
  </section>
</template>

<style scoped>
.filtered-note {
  color: var(--fg-muted);
  font-size: 12px;
  margin: 0 0 12px;
}
.risk-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.risk-row {
  border: 1px solid var(--border-soft);
  border-left: 3px solid var(--border);
  border-radius: 6px;
  padding: 10px 12px;
  background: #ffffff;
}
.risk-row.sev-high { border-left-color: #cf222e; }
.risk-row.sev-medium { border-left-color: #9a6700; }
.risk-row.sev-low { border-left-color: #57606a; }

.head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 6px;
}

.pill {
  font-family: var(--mono);
  font-size: 10px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 999px;
  letter-spacing: 0.04em;
  background: var(--bg-subtle);
  color: var(--fg-muted);
  line-height: 1.4;
}
.pill.sev-high    { background: #ffebe9; color: #82071e; }
.pill.sev-medium  { background: #fff8c5; color: #7d4e00; }
.pill.sev-low     { background: #ddf4ff; color: #0550ae; }

.loc-chip {
  font-family: var(--mono);
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--bg-subtle);
  border: 1px solid var(--border-soft);
  color: var(--fg);
  text-decoration: none;
}
a.loc-chip:hover { background: #eef2f6; color: var(--accent); }
.loc-chip .line { color: var(--fg-muted); }

.cat-tag {
  font-size: 11px;
  color: var(--fg-muted);
  padding: 1px 6px;
  border: 1px solid var(--border-soft);
  border-radius: 3px;
}

.conf {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.conf-track {
  width: 60px;
  height: 6px;
  background: var(--bg-subtle);
  border: 1px solid var(--border-soft);
  border-radius: 3px;
  overflow: hidden;
}
.conf-fill {
  height: 100%;
  background: #57606a;
  transition: width 200ms ease;
}
.conf-fill.sev-high    { background: #cf222e; }
.conf-fill.sev-medium  { background: #9a6700; }
.conf-fill.sev-low     { background: #57606a; }
.conf-text {
  font-size: 11px;
  color: var(--fg-muted);
  min-width: 32px;
  text-align: right;
}

.desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.55;
}
.empty {
  color: var(--fg-muted);
  font-size: 13px;
  margin: 0;
  padding: 12px 0;
  text-align: center;
}
</style>
