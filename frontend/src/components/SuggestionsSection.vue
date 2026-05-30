<script setup>
import { computed } from 'vue'
import { groupByCategory } from '../lib/format.js'
import { prFilesUrl } from '../lib/github.js'

const props = defineProps({
  suggestions: { type: Array, default: () => [] },
  suggestionsError: { type: String, default: '' },
  changes: { type: Object, default: () => ({}) },
})

const groups = computed(() => groupByCategory(props.suggestions || []))
const filesHref = computed(() =>
  prFilesUrl(props.changes?.owner, props.changes?.repo, props.changes?.number),
)
</script>

<template>
  <section class="block">
    <h2>Suggestions</h2>

    <p v-if="suggestionsError" class="section-error">
      本节分析暂时失败：{{ suggestionsError }}
    </p>

    <template v-else>
      <div v-if="groups.length" class="groups">
        <section v-for="g in groups" :key="g.category" class="group">
          <h3 class="cat-head">
            <span class="cat-name">{{ g.category }}</span>
            <span class="cat-count">{{ g.items.length }}</span>
          </h3>
          <ul class="sug-list">
            <li v-for="(s, i) in g.items" :key="i" class="sug-row">
              <div class="head">
                <a
                  v-if="filesHref && s.file"
                  class="loc-chip"
                  :href="filesHref"
                  target="_blank"
                  rel="noopener noreferrer"
                  :title="`Open ${s.file} on GitHub`"
                >
                  {{ s.file }}<span v-if="s.line" class="line">:{{ s.line }}</span>
                </a>
                <span v-else class="loc-chip">
                  {{ s.file || '?' }}<span v-if="s.line" class="line">:{{ s.line }}</span>
                </span>
              </div>
              <p class="text">{{ s.suggestion || '(no suggestion)' }}</p>
            </li>
          </ul>
        </section>
      </div>
      <p v-else class="empty">No suggestions.</p>
    </template>
  </section>
</template>

<style scoped>
.groups {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.cat-head {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--fg-muted);
  margin: 0 0 8px;
}
.cat-name {
  padding: 1px 8px;
  background: var(--bg-subtle);
  border: 1px solid var(--border-soft);
  border-radius: 4px;
  letter-spacing: 0.04em;
}
.cat-count {
  font-variant-numeric: tabular-nums;
  color: var(--fg-muted);
  font-weight: 500;
}
.sug-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.sug-row {
  border: 1px solid var(--border-soft);
  border-left: 3px solid var(--border);
  border-radius: 6px;
  padding: 8px 12px;
  background: #ffffff;
}
.head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 4px;
}
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

.text {
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
