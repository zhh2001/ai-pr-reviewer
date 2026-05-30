<script setup>
import { ref, computed } from 'vue'
import { prFilesUrl } from '../lib/github.js'

const props = defineProps({
  changes: { type: Object, required: true },
})

const showFiles = ref(false)
const fileCount = computed(() => props.changes?.files?.length ?? 0)
const filesUrl = computed(() =>
  prFilesUrl(props.changes?.owner, props.changes?.repo, props.changes?.number),
)

// status → 单字符 badge + 语义色
function statusBadge(s) {
  const k = (s || '').toLowerCase()
  switch (k) {
    case 'added':    return { letter: 'A', cls: 'st-added',    label: 'added' }
    case 'modified': return { letter: 'M', cls: 'st-modified', label: 'modified' }
    case 'removed':
    case 'deleted':  return { letter: 'D', cls: 'st-removed',  label: 'removed' }
    case 'renamed':  return { letter: 'R', cls: 'st-renamed',  label: 'renamed' }
    case 'copied':   return { letter: 'C', cls: 'st-copied',   label: 'copied' }
    default:         return { letter: '?', cls: 'st-other',    label: s || 'unknown' }
  }
}
</script>

<template>
  <section class="block">
    <h2>Changes</h2>
    <div class="meta">
      <div class="title-row">
        <span class="title">{{ changes.title || '(no title)' }}</span>
        <a
          v-if="filesUrl"
          :href="filesUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="ext-link"
          title="Open Files changed on GitHub"
        >View on GitHub →</a>
      </div>
      <div class="sub mono">
        <span class="chip-soft">{{ changes.owner }}/{{ changes.repo }}#{{ changes.number }}</span>
        <span>by @{{ changes.author || '?' }}</span>
        <span class="branch">
          <span class="ref">{{ changes.base_ref }}</span>
          <span class="arr">←</span>
          <span class="ref">{{ changes.head_ref }}</span>
        </span>
      </div>
    </div>
    <button class="toggle" type="button" @click="showFiles = !showFiles">
      <span class="caret">{{ showFiles ? '▾' : '▸' }}</span>
      {{ fileCount }} file{{ fileCount === 1 ? '' : 's' }} changed
    </button>
    <ul v-if="showFiles && fileCount > 0" class="files">
      <li v-for="f in changes.files" :key="f.path" class="file-row">
        <span class="st-badge" :class="statusBadge(f.status).cls" :title="statusBadge(f.status).label">
          {{ statusBadge(f.status).letter }}
        </span>
        <span class="path mono">{{ f.path }}</span>
        <span class="delta">
          <span class="add">+{{ f.additions ?? 0 }}</span>
          <span class="del">−{{ f.deletions ?? 0 }}</span>
        </span>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.meta { display: flex; flex-direction: column; gap: 6px; }
.title-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 16px;
  flex-wrap: wrap;
}
.title {
  font-weight: 600;
  font-size: 15px;
  color: var(--fg);
}
.ext-link {
  font-size: 12px;
  color: var(--accent);
  text-decoration: none;
  white-space: nowrap;
}
.ext-link:hover { text-decoration: underline; }
.sub {
  color: var(--fg-muted);
  font-size: 12px;
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  align-items: center;
}
.chip-soft {
  background: var(--bg-subtle);
  border: 1px solid var(--border-soft);
  border-radius: 4px;
  padding: 1px 6px;
}
.branch { display: inline-flex; gap: 6px; align-items: center; }
.branch .ref {
  background: var(--bg-subtle);
  padding: 1px 6px;
  border-radius: 4px;
}
.branch .arr { color: var(--border); }

.toggle {
  margin-top: 12px;
  background: none;
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 4px 10px;
  font-size: 12px;
  color: var(--fg-muted);
  cursor: pointer;
  font-family: var(--sans);
  align-self: flex-start;
}
.toggle:hover { background: var(--bg-subtle); }
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
  grid-template-columns: 24px 1fr auto;
  gap: 12px;
  align-items: center;
  padding: 6px 10px;
  font-size: 12px;
  border-bottom: 1px solid var(--border-soft);
}
.file-row:last-child { border-bottom: 0; }
.file-row:nth-child(even) { background: #fafbfc; }

.st-badge {
  width: 20px;
  height: 18px;
  border-radius: 4px;
  font-family: var(--mono);
  font-size: 10px;
  font-weight: 700;
  text-align: center;
  line-height: 18px;
  letter-spacing: 0;
}
.st-added    { background: #ddf4dd; color: #1a7f37; }
.st-modified { background: #fff8c5; color: #7d4e00; }
.st-removed  { background: #ffebe9; color: #82071e; }
.st-renamed  { background: #ddf4ff; color: #0550ae; }
.st-copied   { background: #f6f8fa; color: #57606a; border: 1px solid var(--border-soft); }
.st-other    { background: var(--bg-subtle); color: var(--fg-muted); }

.path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.delta { display: inline-flex; gap: 8px; font-variant-numeric: tabular-nums; }
.delta .add { color: #1a7f37; }
.delta .del { color: #cf222e; }
</style>
