<script setup>
import { ref } from 'vue'
import ChangesOverview from './components/ChangesOverview.vue'
import RisksSection from './components/RisksSection.vue'
import SuggestionsSection from './components/SuggestionsSection.vue'

const prUrl = ref('')
const loading = ref(false)
const topError = ref('')
const result = ref(null)

async function review() {
  topError.value = ''
  result.value = null
  loading.value = true
  try {
    const res = await fetch('/api/review', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pr_url: prUrl.value.trim() }),
    })
    if (!res.ok) {
      topError.value = await extractError(res)
      return
    }
    result.value = await res.json()
  } catch (e) {
    topError.value = `Request failed: ${e?.message ?? e}`
  } finally {
    loading.value = false
  }
}

async function extractError(res) {
  const text = await res.text()
  try {
    const body = JSON.parse(text)
    if (body && typeof body.error === 'string') {
      return `${res.status} ${res.statusText}: ${body.error}`
    }
  } catch {
    /* not JSON */
  }
  return `${res.status} ${res.statusText}`
}
</script>

<template>
  <main class="app">
    <header class="header">
      <h1>ai-pr-reviewer</h1>
      <form class="pr-form" @submit.prevent="review">
        <input
          v-model="prUrl"
          class="url-input mono"
          placeholder="owner/repo#123  or  https://github.com/owner/repo/pull/123"
          :disabled="loading"
        />
        <button
          type="submit"
          class="btn"
          :disabled="loading || !prUrl.trim()"
        >
          {{ loading ? 'Reviewing…' : 'Review' }}
        </button>
      </form>
    </header>

    <div v-if="loading" class="loading">
      <span class="dot" aria-hidden="true"></span>
      <span>Fetching diff, running summary / risks / suggestions in parallel…</span>
    </div>

    <div v-if="topError" class="banner error">{{ topError }}</div>

    <section v-if="result && result.changes" class="result">
      <ChangesOverview :changes="result.changes" />

      <section class="block">
        <h2>Summary</h2>
        <p v-if="result.summary" class="summary-text">{{ result.summary }}</p>
        <p v-else-if="result.summary_error" class="section-error">
          本节分析暂时失败：{{ result.summary_error }}
        </p>
        <p v-else class="muted">No summary returned.</p>
      </section>

      <RisksSection
        :risks="result.risks || []"
        :risks-error="result.risks_error || ''"
        :risks-filtered="result.risks_filtered || 0"
      />

      <SuggestionsSection
        :suggestions="result.suggestions || []"
        :suggestions-error="result.suggestions_error || ''"
      />
    </section>
  </main>
</template>

<style>
:root {
  --bg: #ffffff;
  --fg: #1f2328;
  --fg-muted: #57606a;
  --border: #d0d7de;
  --border-soft: #eaeef2;
  --accent: #0969da;
  --sev-high-bg: #ffebe9;
  --sev-high-fg: #82071e;
  --sev-medium-bg: #fff8c5;
  --sev-medium-fg: #7d4e00;
  --sev-low-bg: #ddf4ff;
  --sev-low-fg: #0550ae;
  --error-bg: #ffebe9;
  --error-fg: #82071e;
  --error-border: #ffcecb;
  --mono: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, Consolas, monospace;
  --sans: ui-sans-serif, system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif;
}

* {
  box-sizing: border-box;
}

html,
body {
  margin: 0;
  padding: 0;
}

body {
  font-family: var(--sans);
  background: var(--bg);
  color: var(--fg);
  font-size: 14px;
  line-height: 1.5;
}

.mono {
  font-family: var(--mono);
}

.app {
  max-width: 960px;
  margin: 0 auto;
  padding: 24px 20px 80px;
}

.header h1 {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 12px;
  letter-spacing: -0.01em;
}

.pr-form {
  display: flex;
  gap: 8px;
}

.url-input {
  flex: 1;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  background: #f6f8fa;
  color: var(--fg);
}
.url-input:focus {
  outline: none;
  border-color: var(--accent);
  background: #ffffff;
}
.url-input:disabled {
  opacity: 0.6;
}

.btn {
  padding: 6px 14px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: #f6f8fa;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  font-family: var(--sans);
  color: var(--fg);
}
.btn:hover:not(:disabled) {
  background: #f3f4f6;
}
.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.loading {
  margin-top: 16px;
  padding: 10px 12px;
  border: 1px solid var(--border-soft);
  border-radius: 6px;
  color: var(--fg-muted);
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.loading .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--fg-muted);
  animation: pulse 1.2s infinite ease-in-out;
}
@keyframes pulse {
  0%,
  100% {
    opacity: 0.25;
  }
  50% {
    opacity: 1;
  }
}

.banner {
  margin-top: 16px;
  padding: 10px 12px;
  border-radius: 6px;
  font-size: 13px;
}
.banner.error {
  background: var(--error-bg);
  color: var(--error-fg);
  border: 1px solid var(--error-border);
}

.result {
  margin-top: 24px;
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.block h2 {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--fg-muted);
  margin: 0 0 10px;
  border-bottom: 1px solid var(--border-soft);
  padding-bottom: 6px;
}

.summary-text {
  margin: 0;
  white-space: pre-wrap;
}

.section-error {
  margin: 0;
  color: var(--fg-muted);
  font-size: 13px;
  padding: 8px 10px;
  border-left: 2px solid var(--border);
  background: #f6f8fa;
  border-radius: 0 4px 4px 0;
}

.muted {
  color: var(--fg-muted);
  font-size: 13px;
  margin: 0;
}
</style>
