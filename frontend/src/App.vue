<script setup>
import { ref } from 'vue'
import ChangesOverview from './components/ChangesOverview.vue'
import RisksSection from './components/RisksSection.vue'
import SuggestionsSection from './components/SuggestionsSection.vue'
import SummaryView from './components/SummaryView.vue'
import ResultSummary from './components/ResultSummary.vue'
import Skeleton from './components/Skeleton.vue'

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
  <div class="app-shell">
    <header class="topbar">
      <div class="topbar-inner">
        <div class="brand">
          <span class="brand-name">ai-pr-reviewer</span>
          <span class="brand-tag">Summary · Risks · Suggestions</span>
        </div>
        <form class="pr-form" @submit.prevent="review">
          <input
            v-model="prUrl"
            class="url-input mono"
            placeholder="owner/repo#123  or  https://github.com/owner/repo/pull/123"
            :disabled="loading"
            spellcheck="false"
            autocapitalize="off"
            autocorrect="off"
          />
          <button
            type="submit"
            class="btn primary"
            :disabled="loading || !prUrl.trim()"
          >
            <span class="dot" v-if="loading" aria-hidden="true"></span>
            {{ loading ? 'Reviewing' : 'Review' }}
          </button>
        </form>
      </div>
    </header>

    <main class="app">
      <div v-if="topError" class="banner error">{{ topError }}</div>

      <Skeleton v-if="loading" />

      <section v-else-if="result && result.changes" class="result">
        <ResultSummary :result="result" />
        <ChangesOverview :changes="result.changes" />
        <SummaryView
          :summary="result.summary || ''"
          :summary-error="result.summary_error || ''"
        />
        <RisksSection
          :risks="result.risks || []"
          :risks-error="result.risks_error || ''"
          :risks-filtered="result.risks_filtered || 0"
          :changes="result.changes"
        />
        <SuggestionsSection
          :suggestions="result.suggestions || []"
          :suggestions-error="result.suggestions_error || ''"
          :changes="result.changes"
        />
      </section>

      <div v-else-if="!loading && !topError" class="placeholder">
        <p class="placeholder-title">No review yet.</p>
        <p class="placeholder-hint">
          Paste a public GitHub PR link or <span class="mono">owner/repo#n</span> shorthand above
          and press <kbd>Review</kbd>.
        </p>
      </div>
    </main>
  </div>
</template>

<style>
:root {
  --bg: #ffffff;
  --bg-subtle: #f6f8fa;
  --fg: #1f2328;
  --fg-muted: #57606a;
  --border: #d0d7de;
  --border-soft: #e1e4e8;
  --accent: #0969da;
  --error-bg: #ffebe9;
  --error-fg: #82071e;
  --error-border: #ffcecb;
  --mono: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, Consolas, monospace;
  --sans: ui-sans-serif, system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif;
}

* { box-sizing: border-box; }

html, body {
  margin: 0;
  padding: 0;
  background: var(--bg);
}
body {
  font-family: var(--sans);
  color: var(--fg);
  font-size: 14px;
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
}

.mono { font-family: var(--mono); }

.app-shell { min-height: 100vh; }

/* ── Topbar ────────────────────────────────────────────────── */

.topbar {
  border-bottom: 1px solid var(--border-soft);
  background: #ffffff;
}
.topbar-inner {
  max-width: 1000px;
  margin: 0 auto;
  padding: 12px 24px;
  display: flex;
  align-items: center;
  gap: 24px;
  flex-wrap: wrap;
}
.brand {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
  min-width: max-content;
}
.brand-name {
  font-weight: 600;
  font-size: 14px;
  letter-spacing: -0.01em;
}
.brand-tag {
  font-size: 11px;
  color: var(--fg-muted);
  letter-spacing: 0.02em;
}

.pr-form {
  flex: 1;
  min-width: 320px;
  display: flex;
  gap: 8px;
}
.url-input {
  flex: 1;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  background: var(--bg-subtle);
  color: var(--fg);
  outline: none;
  transition: border-color 120ms, background 120ms;
}
.url-input:focus {
  border-color: var(--accent);
  background: #ffffff;
  box-shadow: 0 0 0 3px rgba(9, 105, 218, 0.12);
}
.url-input:disabled { opacity: 0.65; }
.url-input::placeholder { color: var(--fg-muted); opacity: 0.7; }

.btn {
  padding: 6px 14px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-subtle);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  font-family: var(--sans);
  color: var(--fg);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: background 120ms, border-color 120ms;
}
.btn:hover:not(:disabled) { background: #eef2f6; }
.btn.primary {
  background: #1f2328;
  border-color: #1f2328;
  color: #ffffff;
}
.btn.primary:hover:not(:disabled) {
  background: #2d333b;
  border-color: #2d333b;
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }

.btn .dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.9;
  animation: pulse 1.2s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 1; }
}

/* ── Main column ───────────────────────────────────────────── */

.app {
  max-width: 1000px;
  margin: 0 auto;
  padding: 24px;
}

.banner {
  margin-bottom: 16px;
  padding: 10px 14px;
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.5;
}
.banner.error {
  background: var(--error-bg);
  color: var(--error-fg);
  border: 1px solid var(--error-border);
}

.result {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* ── Section block (shared) ────────────────────────────────── */

.block {
  padding: 16px 18px;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  background: #ffffff;
}
.block h2 {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--fg-muted);
  margin: 0 0 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-soft);
}

.section-error {
  margin: 0;
  color: var(--fg-muted);
  font-size: 13px;
  padding: 8px 10px;
  border-left: 2px solid var(--border);
  background: var(--bg-subtle);
  border-radius: 0 4px 4px 0;
  line-height: 1.5;
}

/* ── Empty placeholder ─────────────────────────────────────── */

.placeholder {
  margin-top: 64px;
  text-align: center;
  color: var(--fg-muted);
}
.placeholder-title {
  font-size: 14px;
  font-weight: 600;
  margin: 0 0 6px;
  color: var(--fg);
}
.placeholder-hint {
  margin: 0;
  font-size: 13px;
}
.placeholder kbd {
  font-family: var(--mono);
  font-size: 11px;
  padding: 1px 6px;
  border: 1px solid var(--border);
  border-bottom-width: 2px;
  border-radius: 4px;
  background: var(--bg-subtle);
  color: var(--fg);
}
</style>
