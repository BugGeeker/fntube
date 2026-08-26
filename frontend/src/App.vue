<template>
  <div class="app-container">
    <!-- 左侧悬浮导航菜单 -->
    <div class="floating-menu">
      <a-menu v-model:selectedKeys="selectedKeys" mode="inline" :inline-collapsed="collapsed" :items="menuItems"
        @click="handleMenuClick" class="menu" />
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
import { ref, watch, h, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  DashboardOutlined,
  VideoCameraOutlined,
  SettingOutlined,
  TranslationOutlined,
  FileTextOutlined,
  ScheduleOutlined,
} from '@ant-design/icons-vue'

const route = useRoute()
const router = useRouter()

const collapsed = ref(false)

function checkCollapsed() {
  collapsed.value = window.innerWidth < 640
}

onMounted(() => {
  checkCollapsed()
  window.addEventListener('resize', checkCollapsed)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', checkCollapsed)
})

const menuItems = [
  {
    key: '/',
    icon: () => h(DashboardOutlined),
    label: '总览',
  },
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
  position: sticky;
  top: 12px;
  align-self: flex-start;
  height: calc(100vh - 24px);
  border-radius: 12px;
  margin: 12px;
  overflow: hidden;

  .menu {
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  }

}

.floating-menu :deep(.ant-menu) {
  height: 100%;
  border-right: none;
}

.main-content {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  padding: 12px;
  min-height: 100vh;
}

</style>
