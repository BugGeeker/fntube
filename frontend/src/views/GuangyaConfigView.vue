<template>
  <div class="guangya-page">
    <a-card title="光鸭云盘设置">
      <a-skeleton v-if="loadingConfig" active :paragraph="{ rows: 2 }" />
      <a-result
        v-else-if="config.bound"
        status="success"
        title="账号已绑定"
        :sub-title="accountDescription"
      >
        <template #extra>
          <a-popconfirm title="确定解绑光鸭云盘账号？" ok-text="解绑" cancel-text="取消" @confirm="handleUnbind">
            <a-button danger :loading="unbinding">
              <template #icon><DisconnectOutlined /></template>
              解绑账号
            </a-button>
          </a-popconfirm>
        </template>
      </a-result>
      <a-empty v-else description="尚未绑定光鸭云盘账号" />
    </a-card>

    <a-card v-if="!config.bound" title="绑定账号">
      <a-tabs v-model:active-key="activeMethod">
        <a-tab-pane key="qr" tab="扫码绑定">
          <div class="qr-panel">
            <template v-if="qrSession">
              <a-qrcode
                :value="qrValue"
                :size="220"
                :status="qrCodeStatus"
                @refresh="startQR"
              />
              <a-typography-text strong>授权码：{{ qrSession.user_code }}</a-typography-text>
              <a-typography-text type="secondary">{{ qrStatusText }}</a-typography-text>
              <a-button v-if="qrLink" :href="qrLink" target="_blank">
                <template #icon><ExportOutlined /></template>
                打开授权页面
              </a-button>
            </template>
            <a-button v-else type="primary" size="large" :loading="startingQR" @click="startQR">
              <template #icon><QrcodeOutlined /></template>
              获取登录二维码
            </a-button>
          </div>
        </a-tab-pane>

        <a-tab-pane key="sms" tab="短信验证码">
          <a-form class="sms-form" layout="vertical" @finish="verifySMS">
            <a-form-item label="手机号" required>
              <a-input
                v-model:value="phone"
                autocomplete="tel"
                placeholder="请输入光鸭云盘账号手机号"
                :disabled="Boolean(smsSessionId)"
              />
            </a-form-item>
            <a-form-item label="短信验证码" required>
              <a-input v-model:value="smsCode" autocomplete="one-time-code" placeholder="请输入短信验证码">
                <template #addonAfter>
                  <a-button
                    type="link"
                    class="send-code-button"
                    :disabled="countdown > 0 || sendingSMS"
                    :loading="sendingSMS"
                    @click="sendSMS"
                  >
                    {{ countdown > 0 ? `${countdown}s` : smsSessionId ? '重新发送' : '发送验证码' }}
                  </a-button>
                </template>
              </a-input>
            </a-form-item>
            <a-button type="primary" html-type="submit" :loading="verifyingSMS" :disabled="!smsSessionId">
              绑定账号
            </a-button>
          </a-form>
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import {
  DisconnectOutlined,
  ExportOutlined,
  QrcodeOutlined,
} from '@ant-design/icons-vue'
import {
  getGuangyaConfig,
  pollGuangyaQR,
  sendGuangyaSMS,
  startGuangyaQR,
  unbindGuangya,
  verifyGuangyaSMS,
  type GuangyaConfig,
  type GuangyaQRSession,
} from '@/api/guangya'

const config = ref<GuangyaConfig>({ bound: false })
const loadingConfig = ref(true)
const unbinding = ref(false)
const activeMethod = ref('qr')

const qrSession = ref<GuangyaQRSession | null>(null)
const startingQR = ref(false)
const qrState = ref<'waiting' | 'expired' | 'denied'>('waiting')
let qrPollTimer: number | undefined

const phone = ref('')
const smsCode = ref('')
const smsSessionId = ref('')
const sendingSMS = ref(false)
const verifyingSMS = ref(false)
const countdown = ref(0)
let countdownTimer: number | undefined

const qrValue = computed(() => qrSession.value?.verification_uri_complete || qrSession.value?.verification_url || '')
const qrLink = computed(() => qrValue.value)
const qrCodeStatus = computed(() => qrState.value === 'expired' ? 'expired' : 'active')
const qrStatusText = computed(() => {
  if (qrState.value === 'expired') return '二维码已过期，请刷新后重试'
  if (qrState.value === 'denied') return '授权已取消，请重新获取二维码'
  return '请使用光鸭云盘扫码并确认授权'
})
const accountDescription = computed(() => {
  const identity = config.value.name || config.value.phone_number || config.value.subject || '光鸭云盘账号'
  const method = config.value.login_method === 'sms' ? '短信验证码' : '扫码'
  return `${identity}，通过${method}绑定`
})

function errorMessage(error: unknown, fallback: string) {
  const candidate = error as { message?: string; response?: { data?: { error?: string } } }
  return candidate.response?.data?.error || candidate.message || fallback
}

async function loadConfig() {
  loadingConfig.value = true
  try {
    const { data } = await getGuangyaConfig()
    config.value = data
  } catch (error) {
    message.error(errorMessage(error, '读取光鸭云盘配置失败'))
  } finally {
    loadingConfig.value = false
  }
}

function stopQRPolling() {
  if (qrPollTimer !== undefined) {
    window.clearTimeout(qrPollTimer)
    qrPollTimer = undefined
  }
}

function scheduleQRPoll() {
  stopQRPolling()
  if (!qrSession.value || qrState.value !== 'waiting') return
  qrPollTimer = window.setTimeout(pollQR, Math.max(qrSession.value.interval, 2) * 1000)
}

async function startQR() {
  stopQRPolling()
  startingQR.value = true
  qrState.value = 'waiting'
  try {
    const { data } = await startGuangyaQR()
    qrSession.value = data
    scheduleQRPoll()
  } catch (error) {
    message.error(errorMessage(error, '获取登录二维码失败'))
  } finally {
    startingQR.value = false
  }
}

async function pollQR() {
  if (!qrSession.value) return
  try {
    const { data } = await pollGuangyaQR(qrSession.value.session_id)
    if (data.state === 'bound') {
      stopQRPolling()
      config.value = data.config || { bound: true, login_method: 'qr' }
      qrSession.value = null
      message.success('光鸭云盘账号绑定成功')
      return
    }
    if (data.state === 'expired' || data.state === 'denied') {
      qrState.value = data.state
      stopQRPolling()
      return
    }
    scheduleQRPoll()
  } catch (error) {
    stopQRPolling()
    message.error(errorMessage(error, '查询扫码状态失败'))
  }
}

function startCountdown() {
  if (countdownTimer !== undefined) window.clearInterval(countdownTimer)
  countdown.value = 60
  countdownTimer = window.setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0 && countdownTimer !== undefined) {
      window.clearInterval(countdownTimer)
      countdownTimer = undefined
    }
  }, 1000)
}

async function sendSMS() {
  if (!phone.value.trim()) {
    message.warning('请输入手机号')
    return
  }
  sendingSMS.value = true
  try {
    const { data } = await sendGuangyaSMS(phone.value.trim(), smsSessionId.value)
    if (data.captcha_required) {
      if (!data.session_id) throw new Error('服务端未返回人机验证会话')
      smsSessionId.value = data.session_id
      if (data.captcha_url) window.open(data.captcha_url, '_blank', 'noopener,noreferrer')
      message.warning('需要先完成人机验证，完成后请重新发送验证码')
      return
    }
    if (!data.session_id) throw new Error('服务端未返回短信登录会话')
    smsSessionId.value = data.session_id
    startCountdown()
    message.success('短信验证码已发送')
  } catch (error) {
    message.error(errorMessage(error, '发送短信验证码失败'))
  } finally {
    sendingSMS.value = false
  }
}

async function verifySMS() {
  if (!smsSessionId.value) {
    message.warning('请先发送短信验证码')
    return
  }
  if (!smsCode.value.trim()) {
    message.warning('请输入短信验证码')
    return
  }
  verifyingSMS.value = true
  try {
    const { data } = await verifyGuangyaSMS(smsSessionId.value, smsCode.value.trim())
    config.value = data
    message.success('光鸭云盘账号绑定成功')
  } catch (error) {
    message.error(errorMessage(error, '绑定失败'))
  } finally {
    verifyingSMS.value = false
  }
}

async function handleUnbind() {
  unbinding.value = true
  try {
    await unbindGuangya()
    config.value = { bound: false }
    qrSession.value = null
    smsSessionId.value = ''
    smsCode.value = ''
    message.success('已解绑光鸭云盘账号')
  } catch (error) {
    message.error(errorMessage(error, '解绑失败'))
  } finally {
    unbinding.value = false
  }
}

onMounted(loadConfig)
onBeforeUnmount(() => {
  stopQRPolling()
  if (countdownTimer !== undefined) window.clearInterval(countdownTimer)
})
</script>

<style scoped lang="scss">
.guangya-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.qr-panel {
  min-height: 330px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  text-align: center;
}

.sms-form {
  max-width: 480px;
}

.send-code-button {
  height: auto;
  padding: 0;
}
</style>
