<script setup>
import { computed } from 'vue'
import { renderMarkdown } from '../lib/markdown.js'
import Skeleton from './Skeleton.vue'

const props = defineProps({
  summary: { type: String, default: '' },
  summaryError: { type: String, default: '' },
  pending: { type: Boolean, default: false },
})

const html = computed(() => renderMarkdown(props.summary || ''))
</script>

<template>
  <section class="block">
    <h2>Summary</h2>
    <p v-if="summaryError" class="section-error">
      本节分析暂时失败：{{ summaryError }}
    </p>
    <div v-else-if="summary" class="md" v-html="html"></div>
    <Skeleton v-else-if="pending" kind="summary" />
    <p v-else class="empty">No summary returned.</p>
  </section>
</template>

<style scoped>
.md {
  font-size: 14px;
  line-height: 1.65;
}
.md :deep(p) { margin: 0 0 0.7em; }
.md :deep(p:last-child) { margin-bottom: 0; }
.md :deep(strong) { font-weight: 600; }
.md :deep(em) { font-style: italic; }
.md :deep(code) {
  background: var(--bg-subtle);
  padding: 1px 4px;
  border-radius: 3px;
  font-family: var(--mono);
  font-size: 12.5px;
}
.md :deep(pre) {
  background: var(--bg-subtle);
  padding: 10px 12px;
  border-radius: 6px;
  overflow-x: auto;
  font-size: 12.5px;
  border: 1px solid var(--border-soft);
}
.md :deep(pre code) { background: transparent; padding: 0; }
.md :deep(ul), .md :deep(ol) { margin: 0 0 0.7em; padding-left: 1.4em; }
.md :deep(li) { margin: 2px 0; }
.md :deep(a) { color: var(--accent); text-decoration: none; }
.md :deep(a:hover) { text-decoration: underline; }
.md :deep(blockquote) {
  margin: 0.5em 0;
  padding: 4px 12px;
  color: var(--fg-muted);
  border-left: 3px solid var(--border);
}
.empty {
  color: var(--fg-muted);
  font-size: 13px;
  margin: 0;
}
</style>
