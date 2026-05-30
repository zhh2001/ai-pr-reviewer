<script setup>
// kind: 'full' 渲染完整结果区骨架；'summary' / 'risks' / 'suggestions' 各自渲染分区
// 内部的占位条。形状刻意贴合该区块完成态的密度，让"骨架→内容"切换时视觉抖动最小。
defineProps({
  kind: { type: String, default: 'full' },
})
</script>

<template>
  <div :class="['skel', `skel-${kind}`]" role="status" aria-busy="true">
    <template v-if="kind === 'full'">
      <div class="block">
        <div class="bar h-meta w-25"></div>
        <div class="bar h-title w-60"></div>
        <div class="bar h-line w-40"></div>
      </div>
      <div class="block">
        <div class="bar h-meta w-15"></div>
        <div class="bar h-line w-90"></div>
        <div class="bar h-line w-95"></div>
        <div class="bar h-line w-70"></div>
      </div>
      <div class="block">
        <div class="bar h-meta w-15"></div>
        <div class="bar h-card"></div>
        <div class="bar h-card"></div>
      </div>
      <div class="block">
        <div class="bar h-meta w-20"></div>
        <div class="bar h-card"></div>
      </div>
    </template>

    <template v-else-if="kind === 'summary'">
      <div class="bar h-line w-95"></div>
      <div class="bar h-line w-90"></div>
      <div class="bar h-line w-70"></div>
    </template>

    <template v-else-if="kind === 'risks'">
      <div class="bar h-card"></div>
      <div class="bar h-card"></div>
    </template>

    <template v-else-if="kind === 'suggestions'">
      <div class="bar h-card"></div>
      <div class="bar h-card"></div>
    </template>
  </div>
</template>

<style scoped>
.skel {
  display: flex;
  flex-direction: column;
}
.skel-full { gap: 24px; }
.skel-summary,
.skel-risks,
.skel-suggestions {
  gap: 8px;
  padding: 2px 0;
}

.block {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
}

.bar {
  background: #eaeef2;
  border-radius: 4px;
  animation: skel-pulse 1.4s ease-in-out infinite;
}
.bar.h-meta { height: 10px; }
.bar.h-title { height: 16px; }
.bar.h-line { height: 12px; }
.bar.h-card { height: 56px; }
.bar.w-15 { width: 15%; }
.bar.w-20 { width: 20%; }
.bar.w-25 { width: 25%; }
.bar.w-40 { width: 40%; }
.bar.w-60 { width: 60%; }
.bar.w-70 { width: 70%; }
.bar.w-80 { width: 80%; }
.bar.w-90 { width: 90%; }
.bar.w-95 { width: 95%; }

@keyframes skel-pulse {
  0%, 100% { opacity: 0.55; }
  50% { opacity: 0.95; }
}
</style>
