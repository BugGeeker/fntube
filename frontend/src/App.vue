<template>
  <div class="app-container">
    <!-- 左侧悬浮导航菜单 -->
    <div class="floating-menu">
      <a-menu
        v-model:selectedKeys="selectedKeys"
        mode="inline"
        theme="dark"
        :items="menuItems"
        @click="handleMenuClick"
      />
    </div>

    <!-- 主内容区 -->
    <div class="main-content">
      <router-view v-slot="{ Component }">
        <keep-alive :include="['MediaView']">
          <component :is="Component" />
        </keep-alive>
      </router-view>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  VideoCameraOutlined,
  SettingOutlined,
  TranslationOutlined,
  FileTextOutlined,
  ScheduleOutlined,
} from '@ant-design/icons-vue'

const route = useRoute()
const router = useRouter()

const menuItems = [
  {
    key: '/media',
    icon: () => h(VideoCameraOutlined),
    label: '媒体浏览',
  },
  {
    key: '/config',
    icon: () => h(SettingOutlined),
    label: '飞牛影视配置',
  },
  {
    key: '/metatube-config',
    icon: () => h(TranslationOutlined),
    label: 'MetaTube 配置',
  },
  {
    key: '/scrape-log',
    icon: () => h(FileTextOutlined),
    label: '刮削记录',
  },
  {
    key: '/scrape-task',
    icon: () => h(ScheduleOutlined),
    label: '刮削计划',
  },
]

const selectedKeys = ref<string[]>([getMenuKey(route.path)])

function getMenuKey(path: string): string {
  if (path.startsWith('/media')) return '/media'
  return path
}

watch(
  () => route.path,
  (newPath) => {
    selectedKeys.value = [getMenuKey(newPath)]
  }
)

function handleMenuClick({ key }: { key: string }) {
  router.push(key)
}
</script>

<style scoped>
.app-container {
  display: flex;
  min-height: 100vh;
  background: #f0f2f5;
}

.floating-menu {
  position: fixed;
  top: 24px;
  left: 24px;
  bottom: 24px;
  width: 200px;
  z-index: 100;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

.floating-menu :deep(.ant-menu) {
  height: 100%;
  border-right: none;
}

.main-content {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  margin-left: 224px;
  padding: 24px;
  min-height: 100vh;
}

@media (max-width: 768px) {
  .floating-menu {
    width: 64px;
  }

  .floating-menu :deep(.ant-menu-item) {
    padding-inline: 0 !important;
    text-align: center;
  }

  .floating-menu :deep(.ant-menu-item .anticon) {
    font-size: 18px;
  }

  .main-content {
    margin-left: 112px;
  }
}
</style>
