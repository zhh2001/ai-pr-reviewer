<script setup>
import { computed } from 'vue'
import { severityCounts } from '../severity.js'

const props = defineProps({
  result: { type: Object, required: true },
})

const counts = computed(() => severityCounts(props.result?.risks || []))
const risksTotal = computed(() => (props.result?.risks || []).length)
const sugTotal = computed(() => (props.result?.suggestions || []).length)
const filesTotal = computed(() => (props.result?.changes?.files || []).length)
const filtered = computed(() => props.result?.risks_filtered || 0)

const distTotal = computed(() => {
  const c = counts.value
  return c.high + c.medium + c.low + c.unknown
})
function pct(n) {
  return distTotal.value === 0 ? 0 : (n / distTotal.value) * 100
}

function pl(n, w) { return n === 1 ? w : w + 's' }
</script>

<template>
  <section class="row">
    <div class="counts">
      <span class="num"><strong>{{ risksTotal }}</strong> {{ pl(risksTotal, 'risk') }}</span>
      <span class="sep">·</span>
      <span class="num"><strong>{{ sugTotal }}</strong> {{ pl(sugTotal, 'suggestion') }}</span>
      <span class="sep">·</span>
      <span class="num"><strong>{{ filesTotal }}</strong> {{ pl(filesTotal, 'file') }}</span>
      <span v-if="filtered > 0" class="filtered">
        <span class="sep">·</span>
        已过滤 {{ filtered }} 条低置信度
      </span>
    </div>
    <div
      v-if="distTotal > 0"
      class="dist"
      :title="`high ${counts.high} · medium ${counts.medium} · low ${counts.low}` + (counts.unknown ? ` · unknown ${counts.unknown}` : '')"
    >
      <span class="seg sev-high" :style="{ width: pct(counts.high) + '%' }"></span>
      <span class="seg sev-medium" :style="{ width: pct(counts.medium) + '%' }"></span>
      <span class="seg sev-low" :style="{ width: pct(counts.low) + '%' }"></span>
      <span class="seg sev-unknown" :style="{ width: pct(counts.unknown) + '%' }"></span>
    </div>
  </section>
</template>

<style scoped>
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 14px;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  background: var(--bg-subtle);
  font-size: 13px;
  flex-wrap: wrap;
}
.counts {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--fg-muted);
  flex-wrap: wrap;
}
.counts .num strong {
  color: var(--fg);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.counts .sep { color: var(--border); }
.filtered { color: var(--fg-muted); }
.dist {
  display: flex;
  height: 6px;
  width: 220px;
  border-radius: 3px;
  overflow: hidden;
  background: #ffffff;
  border: 1px solid var(--border-soft);
}
.dist .seg { display: block; height: 100%; transition: width 200ms ease; }
.dist .sev-high { background: #cf222e; }
.dist .sev-medium { background: #9a6700; }
.dist .sev-low { background: #57606a; }
.dist .sev-unknown { background: #d0d7de; }
</style>
