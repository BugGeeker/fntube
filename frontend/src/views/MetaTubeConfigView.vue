<template>
  <a-form :model="form" layout="vertical" @finish="handleSave">
    <a-card title="MetaTube 服务配置">

      <!-- 服务地址 -->
      <a-form-item label="服务地址" name="host" required>
        <a-input v-model:value="form.host" placeholder="如 http://127.0.0.1:8081" allow-clear />
      </a-form-item>

      <!-- Token -->
      <a-form-item label="API Token" name="token">
        <a-input v-model:value="form.token" placeholder="MetaTube 服务端 Token（非必填）" allow-clear />
      </a-form-item>
    </a-card>
    <a-card title="翻译设置" style="margin-top: 16px">
      <!-- 翻译模式 -->
      <a-form-item label="翻译模式" name="translate_mode">
        <a-select v-model:value="form.translate_mode" :options="translateModeOptions" placeholder="选择翻译模式"
          style="width: 200px" />
      </a-form-item>

      <!-- 翻译引擎 -->
      <a-form-item label="翻译引擎" name="translate_engine">
        <a-select v-model:value="form.translate_engine" :options="translateEngineOptions" placeholder="选择翻译引擎"
          style="width: 200px" @change="onEngineChange" />
      </a-form-item>
    </a-card>
    <!-- 引擎专属配置 -->
    <a-card v-if="form.translate_engine === 'baidu'" title="百度翻译配置" style="margin-top: 16px">
      <a-form-item label="APP ID" name="baidu_app_id">
        <a-input v-model:value="engineConfig.baidu_app_id" placeholder="百度翻译 APP ID" allow-clear />
      </a-form-item>
      <a-form-item label="Secret Key" name="baidu_secret_key">
        <a-input v-model:value="engineConfig.baidu_secret_key" placeholder="百度翻译 Secret Key" allow-clear />
      </a-form-item>
    </a-card>
    <a-card v-if="form.translate_engine === 'deepl'" title="DeepL 配置" style="margin-top: 16px">
      <a-form-item label="API Key" name="deepl_api_key">
        <a-input v-model:value="engineConfig.deepl_api_key" placeholder="DeepL API Key" allow-clear />
      </a-form-item>
    </a-card>
    <a-card v-if="form.translate_engine === 'google'" title="Google 翻译配置" style="margin-top: 16px">
      <a-form-item label="API Key" name="google_api_key">
        <a-input v-model:value="engineConfig.google_api_key" placeholder="Google Cloud Translation API Key"
          allow-clear />
      </a-form-item>
    </a-card>
    <a-card v-if="form.translate_engine === 'openai'" title="OpenAI 配置" style="margin-top: 16px">
      <a-form-item label="API Key" name="openai_api_key">
        <a-input v-model:value="engineConfig.openai_api_key" placeholder="OpenAI API Key" allow-clear />
      </a-form-item>
      <a-form-item label="模型" name="openai_model">
        <a-input v-model:value="engineConfig.openai_model" placeholder="如 gpt-4o-mini" allow-clear />
      </a-form-item>
      <a-form-item label="Base URL" name="openai_base_url">
        <a-input v-model:value="engineConfig.openai_base_url" placeholder="自定义 API 地址（可选）" allow-clear />
      </a-form-item>
    </a-card>
    <a-space style="margin-top: 16px">
      <a-button type="primary" html-type="submit" :loading="saving">保存配置</a-button>
      <a-button :loading="testing" @click="handleTest">测试连接</a-button>
    </a-space>

    <a-alert v-if="testResult" style="margin-top: 16px" :type="testResult.success ? 'success' : 'error'"
      :message="testResult.message" show-icon />

  </a-form>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { message } from 'ant-design-vue'
import { useMetaTubeStore } from '@/stores/metatube'
import {
  TranslateModeOptions,
  TranslateEngineOptions,
  type MetaTubeConfig,
} from '@/api/metatube'

const store = useMetaTubeStore()

const translateModeOptions = TranslateModeOptions
const translateEngineOptions = TranslateEngineOptions

const form = reactive<MetaTubeConfig>({
  host: '',
  token: '',
  translate_mode: 'none',
  translate_engine: 'baidu',
  engine_config: '{}',
})

const engineConfig = reactive<Record<string, string>>({
  baidu_app_id: '',
  baidu_secret_key: '',
  deepl_api_key: '',
  google_api_key: '',
  openai_api_key: '',
  openai_model: '',
  openai_base_url: '',
})

const saving = ref(false)
const testing = ref(false)
const testResult = ref<{ success: boolean; message: string } | null>(null)

onMounted(async () => {
  try {
    await store.fetchConfig()
    if (store.config) {
      Object.assign(form, store.config)
      // 解析 engine_config
      try {
        const ec = JSON.parse(store.config.engine_config || '{}')
        Object.assign(engineConfig, ec)
      } catch { /* ignore */ }
    }
  } catch {
    // 配置尚未保存
  }
})

// 引擎切换时保留之前配置的引擎字段
const engineConfigFields: Record<string, string[]> = {
  baidu: ['baidu_app_id', 'baidu_secret_key'],
  deepl: ['deepl_api_key'],
  google: ['google_api_key'],
  googlefree: [],
  openai: ['openai_api_key', 'openai_model', 'openai_base_url'],
}

function onEngineChange() {
  // 不清空，保留用户已填的配置
}

function buildEngineConfig(): string {
  const fields = engineConfigFields[form.translate_engine] || []
  const cfg: Record<string, string> = {}
  for (const f of fields) {
    if (engineConfig[f]) {
      cfg[f] = engineConfig[f]
    }
  }
  return JSON.stringify(cfg)
}

async function handleSave() {
  if (!form.host) {
    message.warning('请填写服务地址')
    return
  }

  saving.value = true
  try {
    const data = { ...form, engine_config: buildEngineConfig() }
    await store.saveConfigData(data)
    message.success('配置已保存')
  } catch (e: any) {
    message.error('保存失败: ' + (e?.message || ''))
  } finally {
    saving.value = false
  }
}

async function handleTest() {
  if (!form.host) {
    message.warning('请填写服务地址')
    return
  }
  testing.value = true
  testResult.value = null
  try {
    const result = await store.testConnectionData({ ...form })
    const parts = [`连接成功！服务端: ${result.app || '未知'}`]
    if (result.version) {
      parts.push(`版本: ${result.version}`)
    }
    if (form.token && result.token_valid) {
      parts.push('Token 验证通过')
    }
    testResult.value = { success: true, message: parts.join('，') }
  } catch (e: any) {
    testResult.value = {
      success: false,
      message: '连接失败: ' + (e?.message || '请检查服务地址与 Token'),
    }
  } finally {
    testing.value = false
  }
}
</script>