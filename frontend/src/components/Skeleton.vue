<script setup>
// 占位骨架：覆盖结果区主要区块的形状。用静态 opacity 脉冲，不用彩色渐变。
</script>

<template>
  <div class="skel" role="status" aria-label="Loading review">
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
  </div>
</template>

<style scoped>
.skel {
  display: flex;
  flex-direction: column;
  gap: 24px;
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
