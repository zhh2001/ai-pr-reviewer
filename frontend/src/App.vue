<script setup>
import { ref } from 'vue'

const prUrl = ref('')
const loading = ref(false)
const result = ref(null)
const error = ref('')

async function review() {
  error.value = ''
  result.value = null
  loading.value = true
  try {
    const res = await fetch('/api/review', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pr_url: prUrl.value }),
    })
    const text = await res.text()
    try {
      result.value = JSON.parse(text)
    } catch {
      result.value = text
    }
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main>
    <h1>ai-pr-reviewer</h1>
    <div class="row">
      <input
        v-model="prUrl"
        placeholder="https://github.com/owner/repo/pull/123"
      />
      <button :disabled="loading || !prUrl" @click="review">
        {{ loading ? 'Reviewing…' : 'Review' }}
      </button>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
    <pre v-if="result">{{
      typeof result === 'string' ? result : JSON.stringify(result, null, 2)
    }}</pre>
  </main>
</template>

<style>
body {
  font-family: system-ui, sans-serif;
  margin: 0;
}
main {
  max-width: 720px;
  margin: 2rem auto;
  padding: 0 1rem;
}
.row {
  display: flex;
  gap: 0.5rem;
}
input {
  flex: 1;
  padding: 0.5rem;
  font-size: 1rem;
}
button {
  padding: 0.5rem 1rem;
  font-size: 1rem;
  cursor: pointer;
}
button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.error {
  color: #c00;
}
pre {
  background: #f5f5f5;
  padding: 1rem;
  border-radius: 4px;
  overflow-x: auto;
}
</style>
