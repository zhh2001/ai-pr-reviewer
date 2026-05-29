<script setup>
defineProps({
  suggestions: { type: Array, default: () => [] },
  suggestionsError: { type: String, default: '' },
})
</script>

<template>
  <section class="block">
    <h2>Suggestions</h2>

    <p v-if="suggestionsError" class="section-error">
      本节分析暂时失败：{{ suggestionsError }}
    </p>

    <template v-else>
      <ul v-if="suggestions?.length" class="sug-list">
        <li v-for="(s, i) in suggestions" :key="i" class="sug-row">
          <div class="head">
            <span class="loc mono">
              {{ s.file || '?' }}<span v-if="s.line">:{{ s.line }}</span>
            </span>
            <span class="cat">{{ s.category || '—' }}</span>
          </div>
          <p class="text">{{ s.suggestion || '(no suggestion)' }}</p>
        </li>
      </ul>
      <p v-else class="muted">No suggestions.</p>
    </template>
  </section>
</template>

<style scoped>
.sug-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.sug-row {
  border: 1px solid var(--border-soft);
  border-left: 3px solid var(--border);
  border-radius: 4px;
  padding: 10px 12px;
  margin-bottom: 8px;
}
.head {
  display: flex;
  gap: 10px;
  align-items: center;
  font-size: 12px;
  margin-bottom: 6px;
  flex-wrap: wrap;
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
.text {
  margin: 0;
  font-size: 13px;
}
</style>
