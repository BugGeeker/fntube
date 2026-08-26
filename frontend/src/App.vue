<template>
  <a-config-provider :theme="themeConfig">
    <div class="app-container">
      <!-- 左侧悬浮导航菜单 -->
      <div class="floating-menu">
        <a-menu v-model:selectedKeys="selectedKeys" mode="inline" :inline-collapsed="collapsed" :items="menuItems"
          @click="handleMenuClick" class="menu" />
        <div class="menu-footer" :class="{ 'menu-footer-collapsed': collapsed }">
          <a-tooltip :title="uiStore.hideImages ? '当前已隐藏图片，点击显示图片' : '点击隐藏所有图片'">
            <a-switch
              v-model:checked="hideImagesChecked"
              checked-children="隐图"
              un-checked-children="显图"
            />
          </a-tooltip>
          <a-tooltip :title="uiStore.darkMode ? '当前为深色模式，点击切换浅色' : '点击切换深色模式'">
            <a-button :type="uiStore.darkMode ? 'primary' : 'default'" shape="circle" size="small" @click="uiStore.toggleDarkMode()">
              <template #icon>
                <component :is="uiStore.darkMode ? BulbFilled : BulbOutlined" />
              </template>
            </a-button>
          </a-tooltip>
        </div>
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
  padding: 12px 10px;
  border-top: 1px solid v-bind('token.colorBorderSecondary');
  background: v-bind('token.colorBgContainer');
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.menu-footer-collapsed {
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
</style>
