<script setup>
import { computed } from 'vue'
import { sortBySeverity, severityClass } from '../severity.js'

const props = defineProps({
  risks: { type: Array, default: () => [] },
  risksError: { type: String, default: '' },
  risksFiltered: { type: Number, default: 0 },
})

const sorted = computed(() => sortBySeverity(props.risks ?? []))

function formatConfidence(c) {
  if (c == null || Number.isNaN(c)) return ''
  return `${Math.round(c * 100)}%`
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
        已过滤 {{ risksFiltered }} 条低置信度项
      </p>

      <ul v-if="sorted.length" class="risk-list">
        <li
          v-for="(r, i) in sorted"
          :key="i"
          class="risk-row"
          :class="severityClass(r.severity)"
        >
          <div class="head">
            <span class="sev-badge">{{ (r.severity || '?').toUpperCase() }}</span>
            <span class="loc mono">
              {{ r.file || '?' }}<span v-if="r.line">:{{ r.line }}</span>
            </span>
            <span class="cat">{{ r.category || '—' }}</span>
            <span class="conf mono">{{ formatConfidence(r.confidence) }}</span>
          </div>
          <p class="desc">{{ r.description || '(no description)' }}</p>
        </li>
      </ul>
      <p v-else class="muted">No risks identified.</p>
    </template>
  </section>
</template>

<style scoped>
.filtered-note {
  color: var(--fg-muted);
  font-size: 12px;
  margin: 0 0 10px;
}
.risk-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.risk-row {
  border: 1px solid var(--border-soft);
  border-left: 3px solid var(--border);
  border-radius: 4px;
  padding: 10px 12px;
  margin-bottom: 8px;
}
.risk-row.sev-high {
  border-left-color: #cf222e;
}
.risk-row.sev-medium {
  border-left-color: #9a6700;
}
.risk-row.sev-low {
  border-left-color: #57606a;
}

.head {
  display: flex;
  gap: 10px;
  align-items: center;
  font-size: 12px;
  margin-bottom: 6px;
  flex-wrap: wrap;
}
.sev-badge {
  font-family: var(--mono);
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 3px;
  letter-spacing: 0.04em;
  background: #f6f8fa;
  color: var(--fg-muted);
}
.sev-high .sev-badge {
  background: var(--sev-high-bg);
  color: var(--sev-high-fg);
}
.sev-medium .sev-badge {
  background: var(--sev-medium-bg);
  color: var(--sev-medium-fg);
}
.sev-low .sev-badge {
  background: var(--sev-low-bg);
  color: var(--sev-low-fg);
}
.loc {
  font-size: 12px;
}
.cat {
  color: var(--fg-muted);
  font-size: 11px;
  padding: 1px 6px;
  border: 1px solid var(--border-soft);
  border-radius: 3px;
}
.conf {
  margin-left: auto;
  color: var(--fg-muted);
  font-size: 12px;
}
.desc {
  margin: 0;
  font-size: 13px;
}
</style>
