import { defineStore } from 'pinia'
import { TrimApp } from '@trimjs/web-app'
import { computed, ref, watch } from 'vue'

const STORAGE_KEY = 'fntube_hide_images'
const THEME_KEY = 'fntube_theme_mode'
const LEGACY_DARK_KEY = 'fntube_dark_mode'

export type ThemeMode = 'system' | 'dark' | 'light'
type ResolvedTheme = Exclude<ThemeMode, 'system'>

function getBrowserTheme(): ResolvedTheme {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function getStoredThemeMode(): ThemeMode {
  const storedTheme = localStorage.getItem(THEME_KEY)
  if (storedTheme === 'system' || storedTheme === 'dark' || storedTheme === 'light') {
    return storedTheme
  }

  const legacyDark = localStorage.getItem(LEGACY_DARK_KEY)
  if (legacyDark === 'true') return 'dark'
  if (legacyDark === 'false') return 'light'
  return 'system'
}

/**
 * 全局 UI 相关状态，目前主要用于「隐藏图片」开关和主题切换。
 *
 * 该状态持久化到 localStorage，刷新页面后仍生效；
 * 同时在 window 上派发 `fntube:hide-images-change` 事件，
 * 便于未使用 Pinia 的工具函数（如 proxyImage）感知变化。
 */
export const useUiStore = defineStore('ui', () => {
  const stored = localStorage.getItem(STORAGE_KEY)
  const hideImages = ref(stored === 'true')

  const themeMode = ref<ThemeMode>(getStoredThemeMode())
  const systemTheme = ref<ResolvedTheme>(getBrowserTheme())
  const darkMode = computed(() =>
    themeMode.value === 'system' ? systemTheme.value === 'dark' : themeMode.value === 'dark'
  )

  let platformThemeAvailable = false
  let initialized = false
  let refreshSequence = 0
  let trimApp: TrimApp | null = null
  let mediaQuery: MediaQueryList | null = null

  watch(
    hideImages,
    (val) => {
      localStorage.setItem(STORAGE_KEY, String(val))
      window.dispatchEvent(new CustomEvent('fntube:hide-images-change', { detail: val }))
    }
  )

  watch(
    [themeMode, systemTheme],
    () => {
      localStorage.setItem(THEME_KEY, themeMode.value)
      localStorage.removeItem(LEGACY_DARK_KEY)
      document.documentElement.classList.toggle('dark', darkMode.value)
    },
    { immediate: true }
  )

  function setHideImages(val: boolean) {
    hideImages.value = val
  }

  function toggleHideImages() {
    hideImages.value = !hideImages.value
  }

  async function refreshPlatformTheme() {
    const sequence = ++refreshSequence
    trimApp ??= new TrimApp()

    try {
      const config = await trimApp.getPlatformConfig()
      if (sequence !== refreshSequence) return
      platformThemeAvailable = true
      systemTheme.value = config.theme
    } catch {
      if (sequence !== refreshSequence) return
      platformThemeAvailable = false
      systemTheme.value = getBrowserTheme()
    }
  }

  function handleBrowserThemeChange(event: MediaQueryListEvent) {
    if (!platformThemeAvailable) {
      systemTheme.value = event.matches ? 'dark' : 'light'
    }
  }

  function handleWindowFocus() {
    if (themeMode.value === 'system') {
      void refreshPlatformTheme()
    }
  }

  function handleVisibilityChange() {
    if (document.visibilityState === 'visible') {
      handleWindowFocus()
    }
  }

  function initializePlatformTheme() {
    if (initialized) return
    initialized = true
    mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    mediaQuery.addEventListener('change', handleBrowserThemeChange)
    window.addEventListener('focus', handleWindowFocus)
    document.addEventListener('visibilitychange', handleVisibilityChange)
    void refreshPlatformTheme()
  }

  function disposePlatformTheme() {
    if (!initialized) return
    initialized = false
    refreshSequence++
    mediaQuery?.removeEventListener('change', handleBrowserThemeChange)
    window.removeEventListener('focus', handleWindowFocus)
    document.removeEventListener('visibilitychange', handleVisibilityChange)
    mediaQuery = null
    trimApp = null
  }

  function setThemeMode(mode: ThemeMode) {
    themeMode.value = mode
    if (mode === 'system') {
      void refreshPlatformTheme()
    }
  }

  function cycleThemeMode() {
    const modes: ThemeMode[] = ['system', 'dark', 'light']
    const currentIndex = modes.indexOf(themeMode.value)
    setThemeMode(modes[(currentIndex + 1) % modes.length])
  }

  function setDarkMode(val: boolean) {
    setThemeMode(val ? 'dark' : 'light')
  }

  function toggleDarkMode() {
    setDarkMode(!darkMode.value)
  }

  return {
    hideImages,
    setHideImages,
    toggleHideImages,
    themeMode,
    systemTheme,
    darkMode,
    setThemeMode,
    cycleThemeMode,
    setDarkMode,
    toggleDarkMode,
    initializePlatformTheme,
    disposePlatformTheme,
  }
})
