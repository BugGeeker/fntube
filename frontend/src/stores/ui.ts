import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

const STORAGE_KEY = 'fntube_hide_images'
const DARK_KEY = 'fntube_dark_mode'

/**
 * 全局 UI 相关状态，目前主要用于「隐藏图片」开关和「深色模式」切换。
 *
 * 该状态持久化到 localStorage，刷新页面后仍生效；
 * 同时在 window 上派发 `fntube:hide-images-change` 事件，
 * 便于未使用 Pinia 的工具函数（如 proxyImage）感知变化。
 */
export const useUiStore = defineStore('ui', () => {
  const stored = localStorage.getItem(STORAGE_KEY)
  const hideImages = ref(stored === 'true')

  const storedDark = localStorage.getItem(DARK_KEY)
  const darkMode = ref(storedDark === 'true')

  watch(
    hideImages,
    (val) => {
      localStorage.setItem(STORAGE_KEY, String(val))
      window.dispatchEvent(new CustomEvent('fntube:hide-images-change', { detail: val }))
    }
  )

  watch(
    darkMode,
    (val) => {
      localStorage.setItem(DARK_KEY, String(val))
      document.documentElement.classList.toggle('dark', val)
    },
    { immediate: true }
  )

  function setHideImages(val: boolean) {
    hideImages.value = val
  }

  function toggleHideImages() {
    hideImages.value = !hideImages.value
  }

  function setDarkMode(val: boolean) {
    darkMode.value = val
  }

  function toggleDarkMode() {
    darkMode.value = !darkMode.value
  }

  return {
    hideImages,
    setHideImages,
    toggleHideImages,
    darkMode,
    setDarkMode,
    toggleDarkMode,
  }
})
