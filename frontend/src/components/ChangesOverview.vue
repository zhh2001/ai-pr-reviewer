<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  changes: { type: Object, required: true },
})

const showFiles = ref(false)
const fileCount = computed(() => props.changes?.files?.length ?? 0)
</script>

<template>
  <section class="block">
    <h2>Changes</h2>
    <div class="meta">
      <div class="title">{{ changes.title || '(no title)' }}</div>
      <div class="sub mono">
        <span>{{ changes.owner }}/{{ changes.repo }}#{{ changes.number }}</span>
        <span class="sep">·</span>
        <span>@{{ changes.author || '?' }}</span>
        <span class="sep">·</span>
        <span>{{ changes.base_ref }} → {{ changes.head_ref }}</span>
      </div>
    </div>
    <button class="toggle" type="button" @click="showFiles = !showFiles">
      <span class="caret">{{ showFiles ? '▾' : '▸' }}</span>
      {{ fileCount }} file{{ fileCount === 1 ? '' : 's' }} changed
    </button>
    <ul v-if="showFiles && fileCount > 0" class="files">
      <li v-for="f in changes.files" :key="f.path" class="file-row">
        <span class="path mono">{{ f.path }}</span>
        <span class="add mono">+{{ f.additions ?? 0 }}</span>
        <span class="del mono">-{{ f.deletions ?? 0 }}</span>
        <span class="status">{{ f.status }}</span>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.meta .title {
  font-weight: 600;
  font-size: 15px;
  margin-bottom: 4px;
}
.meta .sub {
  color: var(--fg-muted);
  font-size: 12px;
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.meta .sub .sep {
  color: var(--border);
}
.toggle {
  margin-top: 10px;
  background: none;
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 4px 10px;
  font-size: 12px;
  color: var(--fg-muted);
  cursor: pointer;
  font-family: var(--sans);
}
.toggle:hover {
  background: #f6f8fa;
}
.caret {
  display: inline-block;
  width: 10px;
  font-family: var(--mono);
}
.files {
  list-style: none;
  margin: 8px 0 0;
  padding: 0;
  border: 1px solid var(--border-soft);
  border-radius: 6px;
  overflow: hidden;
}
.file-row {
  display: grid;
  grid-template-columns: 1fr auto auto auto;
  gap: 12px;
  align-items: center;
  padding: 6px 10px;
  font-size: 12px;
  border-bottom: 1px solid var(--border-soft);
}
.file-row:last-child {
  border-bottom: 0;
}
.file-row .path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-row .add {
  color: #1a7f37;
}
.file-row .del {
  color: #cf222e;
}
.file-row .status {
  color: var(--fg-muted);
  font-size: 11px;
  min-width: 72px;
  text-align: right;
}
</style>
