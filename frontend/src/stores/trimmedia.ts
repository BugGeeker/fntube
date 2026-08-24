import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  getConfig,
  saveConfig,
  testConnection,
  getLibraries,
  getItems,
  getItem,
  getSeasons,
  getEpisodes,
  getPersons,
  getPlayURL,
  getResume,
  getLatest,
  getStatistics,
  refreshLibraries,
  searchMedia,
  type TrimMediaConfig,
  type Library,
  type MediaServerItem,
  type PlayItem,
  type MediaStatistics,
  type SeasonItem,
  type Person,
} from '@/api/trimmedia'

export const useTrimMediaStore = defineStore('trimmedia', () => {
  const config = ref<TrimMediaConfig | null>(null)
  const libraries = ref<Library[]>([])
  const items = ref<MediaServerItem[]>([])
  const itemsTotal = ref(0)
  const currentItem = ref<MediaServerItem | null>(null)
  const seasons = ref<SeasonItem[]>([])
  const episodes = ref<SeasonItem[]>([])
  const persons = ref<Person[]>([])
  const resumeList = ref<PlayItem[]>([])
  const latestList = ref<PlayItem[]>([])
  const statistics = ref<MediaStatistics | null>(null)
  const searchResults = ref<MediaServerItem[]>([])
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

  async function saveConfigData(data: TrimMediaConfig) {
    const { data: result } = await saveConfig(data)
    config.value = data
    return result
  }

  async function testConnectionData(data: Partial<TrimMediaConfig>) {
    const { data: result } = await testConnection(data)
    connected.value = result.status === 'ok'
    return result
  }

  async function fetchLibraries() {
    loading.value = true
    try {
      const { data } = await getLibraries()
      libraries.value = data || []
      connected.value = true
    } finally {
      loading.value = false
    }
  }

  async function fetchItems(libraryId: string, start = 0, limit = 20) {
    loading.value = true
    try {
      const { data } = await getItems(libraryId, start, limit)
      items.value = data?.items || []
      itemsTotal.value = data?.total || 0
    } finally {
      loading.value = false
    }
  }

  async function fetchItem(itemId: string) {
    loading.value = true
    try {
      const { data } = await getItem(itemId)
      currentItem.value = data
    } finally {
      loading.value = false
    }
  }

  async function fetchSeasons(tvId: string) {
    const { data } = await getSeasons(tvId)
    seasons.value = data || []
  }

  async function fetchEpisodes(seasonId: string) {
    const { data } = await getEpisodes(seasonId)
    episodes.value = data || []
  }

  async function fetchPersons(itemId: string) {
    const { data } = await getPersons(itemId)
    persons.value = data || []
  }

  async function fetchPlayURL(itemId: string) {
    const { data } = await getPlayURL(itemId)
    return data.url
  }

  async function fetchResume(num = 12) {
    const { data } = await getResume(num)
    resumeList.value = data || []
  }

  async function fetchLatest(num = 20) {
    const { data } = await getLatest(num)
    latestList.value = data || []
  }

  async function fetchStatistics() {
    const { data } = await getStatistics()
    statistics.value = data
  }

  async function refresh(paths?: string[]) {
    const { data } = await refreshLibraries(paths)
    return data.success
  }

  async function search(q: string) {
    const { data } = await searchMedia(q)
    searchResults.value = data || []
  }

  return {
    config,
    libraries,
    items,
    itemsTotal,
    currentItem,
    seasons,
    episodes,
    persons,
    resumeList,
    latestList,
    statistics,
    searchResults,
    loading,
    connected,
    fetchConfig,
    saveConfigData,
    testConnectionData,
    fetchLibraries,
    fetchItems,
    fetchItem,
    fetchSeasons,
    fetchEpisodes,
    fetchPersons,
    fetchPlayURL,
    fetchResume,
    fetchLatest,
    fetchStatistics,
    refresh,
    search,
  }
})
