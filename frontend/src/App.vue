<template>
  <a-config-provider :theme="themeConfig">
    <div class="app-container">
      <!-- 左侧悬浮导航菜单 -->
      <div class="floating-menu">
        <a-menu v-model:selectedKeys="selectedKeys" mode="inline" :inline-collapsed="collapsed" :items="menuItems"
          @click="handleMenuClick" class="menu" />
        <div class="menu-footer" :class="{ 'menu-footer-collapsed': collapsed }" role="contentinfo">
          <div class="menu-actions">
            <a-switch v-model:checked="hideImagesChecked" checked-children="隐图" un-checked-children="显图" />
            <a-button :type="uiStore.darkMode ? 'primary' : 'default'" shape="circle" size="small"
              @click="uiStore.toggleDarkMode()">
              <template #icon>
                <component :is="uiStore.darkMode ? BulbFilled : BulbOutlined" />
              </template>
            </a-button>
          </div>
          <span v-if="appVersion" class="app-version">v{{ appVersion }}</span>
        </div>
      </div>

      <!-- 移动端底部导航 -->
      <div class="mobile-tabbar">
        <button v-for="item in mobileTabItems" :key="item.key" class="mobile-tab"
          :class="{ active: mobileSelectedKey === item.key }" @click="handleMenuClick({ key: item.key })">
          <component :is="item.icon" />
          <span>{{ item.label }}</span>
        </button>
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
  </a-config-provider>
</template>

<script setup lang="ts">
import { ref, watch, h, onMounted, onBeforeUnmount, computed } from 'vue'
import request from '@/api/request'
import { useRoute, useRouter } from 'vue-router'
import {
  DashboardOutlined,
  VideoCameraOutlined,
  SettingOutlined,
  TranslationOutlined,
  FileTextOutlined,
  ScheduleOutlined,
  BulbOutlined,
  BulbFilled,
} from '@ant-design/icons-vue'
import { theme } from 'ant-design-vue'
import { useUiStore } from '@/stores/ui'

const appVersion = ref('')
const route = useRoute()
const router = useRouter()
const uiStore = useUiStore()

const hideImagesChecked = computed({
  get: () => uiStore.hideImages,
  set: (val: boolean) => uiStore.setHideImages(val),
})

const { token } = theme.useToken()

const themeConfig = computed(() => ({
  algorithm: uiStore.darkMode ? theme.darkAlgorithm : theme.defaultAlgorithm,
}))

const collapsed = ref(false)

function checkCollapsed() {
  collapsed.value = window.innerWidth < 640
}

onMounted(async () => {
  checkCollapsed()
  window.addEventListener('resize', checkCollapsed)
  try {
    const { data } = await request.get<{ version: string }>('/version')
    appVersion.value = data.version
  } catch {
    // 开发环境未注入版本变量时不显示版本号
  }
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

const mobileTabItems = [
  { key: '/', icon: DashboardOutlined, label: '总览' },
  { key: '/media', icon: VideoCameraOutlined, label: '媒体浏览' },
  { key: '/scrape-log', icon: FileTextOutlined, label: '刮削记录' },
  { key: '/scrape-task', icon: ScheduleOutlined, label: '刮削计划' },
  { key: '/settings', icon: SettingOutlined, label: '配置' },
]

const mobileSelectedKey = computed(() => {
  if (['/settings', '/config', '/metatube-config'].includes(route.path)) return '/settings'
  return getMenuKey(route.path)
})

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

<style scoped lang="scss">
.app-container {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: v-bind('token.colorBgLayout');
}

.floating-menu {
  position: sticky;
  top: 12px;
  align-self: flex-start;
  height: calc(100vh - 24px);
  border-radius: 12px;
  margin: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: v-bind('token.colorBgContainer');

  .menu {
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
    flex: 1;
  }

}

.menu-footer {
  padding: 10px;
  border-top: 1px solid v-bind('token.colorBorderSecondary');
  background: v-bind('token.colorBgContainer');
  text-align: center;
}

.menu-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.app-version {
  display: block;
  margin-top: 8px;
  color: v-bind('token.colorTextTertiary');
  font-size: 12px;
  line-height: 1;
}

.menu-footer-collapsed .menu-actions {
  flex-direction: column;
  gap: 10px;
}

.floating-menu :deep(.ant-menu) {
  height: 100%;
  border-right: none;
}

.main-content {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 12px;
  height: 100vh;
}

.mobile-tabbar {
  display: none;
}

@media (max-width: 639px) {
  .floating-menu {
    display: none;
  }

  .main-content {
    height: 100vh;
    padding: 8px 8px calc(72px + env(safe-area-inset-bottom));
  }

  .mobile-tabbar {
    position: fixed;
    z-index: 1000;
    right: 0;
    bottom: 0;
    left: 0;
    display: flex;
    padding: 6px 4px env(safe-area-inset-bottom);
    background: v-bind('token.colorBgContainer');
    border-top: 1px solid v-bind('token.colorBorderSecondary');
    box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.08);
  }

  .mobile-tab {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    padding: 3px 0;
    color: v-bind('token.colorTextSecondary');
    font-size: 11px;
    background: transparent;
    border: 0;
  }

  .mobile-tab :deep(svg) {
    font-size: 20px;
  }

  .mobile-tab.active {
    color: v-bind('token.colorPrimary');
  }
}
</style>
