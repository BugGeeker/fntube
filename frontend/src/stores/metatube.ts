import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getConfig, saveConfig, testConnection, searchMovies, type MetaTubeConfig, type MovieSearchResult } from '@/api/metatube'

export const useMetaTubeStore = defineStore('metatube', () => {
  const config = ref<MetaTubeConfig | null>(null)
  const loading = ref(false)
  const connected = ref(false)
  const searchResults = ref<MovieSearchResult[]>([])

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

  async function searchMoviesData(q: string) {
    loading.value = true
    try {
      const { data } = await searchMovies(q)
      searchResults.value = data.data || []
    } finally {
      loading.value = false
    }
  }

  return {
    config,
    loading,
    connected,
    searchResults,
    fetchConfig,
    saveConfigData,
    testConnectionData,
    searchMoviesData,
  }
})