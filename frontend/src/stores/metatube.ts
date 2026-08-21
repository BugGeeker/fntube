import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getConfig, saveConfig, testConnection, type MetaTubeConfig } from '@/api/metatube'

export const useMetaTubeStore = defineStore('metatube', () => {
  const config = ref<MetaTubeConfig | null>(null)
  const loading = ref(false)
  const connected = ref(false)

  async function fetchConfig() {
    loading.value = true
    try {
      const { data } = await getConfig()
      config.value = data
    } finally {
      loading.value = false
    }
  }

  async function saveConfigData(data: MetaTubeConfig) {
    const { data: result } = await saveConfig(data)
    config.value = data
    return result
  }

  async function testConnectionData(data: Partial<MetaTubeConfig>) {
    const { data: result } = await testConnection(data)
    connected.value = result.status === 'ok'
    return result
  }

  return {
    config,
    loading,
    connected,
    fetchConfig,
    saveConfigData,
    testConnectionData,
  }
})