<template>
  <div style="text-align: center; padding: 48px">
    <h1>欢迎使用 FnTube</h1>
    <p>飞牛应用包 - Go + Hertz + SQLite 后端，Vue3 + Ant Design Vue 前端</p>
    <a-space direction="vertical" size="large">
      <a-button type="primary" @click="$router.push('/media')">媒体浏览</a-button>
      <a-button @click="$router.push('/config')">飞牛影视配置</a-button>
      <a-button @click="$router.push('/tasks')">任务管理</a-button>
    </a-space>
  </div>
</template>

<script setup lang="ts">
</script>
