<template>
  <section class="chat-page">
    <header class="chat-page__header">
      <div class="chat-page__identity">
        <h1 class="chat-page__title">林薇</h1>
        <p class="chat-page__subtitle">KnoX 智能助手 · 独立书店主理人</p>
      </div>
      <div class="chat-page__status" :class="{ 'chat-page__status--active': isStreaming }">
        <span class="chat-page__status-dot"></span>
        {{ isStreaming ? '回答中' : '在线' }}
      </div>
    </header>

    <main ref="messageListRef" class="chat-page__messages">
      <article
        v-for="message in messages"
        :key="message.id"
        class="chat-message"
        :class="`chat-message--${message.role}`"
      >
        <div class="chat-message__bubble">
          <span class="chat-message__text">{{ message.content }}</span>
        </div>
      </article>

      <div v-if="isStreaming" class="chat-message chat-message--assistant">
        <div class="chat-message__bubble chat-message__bubble--typing">
          <span class="chat-message__text">正在思考</span>
          <span class="chat-message__ellipsis" aria-hidden="true"></span>
        </div>
      </div>
    </main>

    <footer class="chat-page__composer">
      <form class="chat-composer" @submit.prevent="handleSend">
        <textarea
          v-model="draft"
          class="chat-composer__input"
          rows="3"
          :placeholder="props.placeholder"
          @keydown.enter.exact.prevent="handleSend"
        ></textarea>
        <div class="chat-composer__actions">
          <span class="chat-composer__hint">Enter 发送，Shift+Enter 换行</span>
          <button
            type="submit"
            class="chat-composer__button"
            :disabled="isStreaming || !draft.trim()"
          >
            {{ isStreaming ? '生成中…' : '发送' }}
          </button>
        </div>
      </form>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref } from 'vue'

type ChatRole = 'user' | 'assistant'

interface ChatMessage {
  id: string
  role: ChatRole
  content: string
}

interface ChatRequestBody {
  question: string
  sessionId?: string
}

interface ChatEventPayload {
  content?: string
  sessionId?: string
  error?: string
}

interface ChatPageProps {
  apiBaseUrl: string
  initialSessionId?: string
  placeholder?: string
}

const props = withDefaults(defineProps<ChatPageProps>(), {
  apiBaseUrl: '/api/v1',
  initialSessionId: '',
  placeholder: '输入你的问题，Enter 发送，Shift+Enter 换行',
})

const emit = defineEmits<{
  (event: 'session-change', sessionId: string): void
}>()

const messages = ref<ChatMessage[]>([])
const draft = ref('')
const sessionId = ref(props.initialSessionId)
const isStreaming = ref(false)
const messageListRef = ref<HTMLElement | null>(null)

let abortController: AbortController | null = null

function createMessageId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `msg-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function parseChatEventPayload(raw: string): ChatEventPayload | null {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return null
  }
  if (!isRecord(parsed)) {
    return null
  }
  const payload: ChatEventPayload = {}
  if (typeof parsed.content === 'string') {
    payload.content = parsed.content
  }
  if (typeof parsed.sessionId === 'string') {
    payload.sessionId = parsed.sessionId
  }
  if (typeof parsed.error === 'string') {
    payload.error = parsed.error
  }
  return payload
}

function getLastAssistantMessage(): ChatMessage | null {
  for (let i = messages.value.length - 1; i >= 0; i--) {
    const message = messages.value[i]
    if (message.role === 'assistant') {
      return message
    }
  }
  return null
}

async function scrollToBottom(): Promise<void> {
  await nextTick()
  const element = messageListRef.value
  if (element) {
    element.scrollTop = element.scrollHeight
  }
}

function applyPayload(payload: ChatEventPayload): void {
  if (payload.error) {
    const last = getLastAssistantMessage()
    if (last && !last.content) {
      last.content = payload.error
    }
    return
  }
  if (payload.sessionId && sessionId.value !== payload.sessionId) {
    sessionId.value = payload.sessionId
    emit('session-change', payload.sessionId)
  }
  if (payload.content) {
    const last = getLastAssistantMessage()
    if (last) {
      last.content += payload.content
    }
    void scrollToBottom()
  }
}

function handleChatError(error: unknown): void {
  const last = getLastAssistantMessage()
  const reason =
    error instanceof DOMException && error.name === 'AbortError'
      ? '请求已取消'
      : '请求失败，请检查网络后重试'
  if (last && !last.content) {
    last.content = reason
  }
}

async function sendChat(question: string): Promise<void> {
  const controller = new AbortController()
  abortController = controller
  isStreaming.value = true

  try {
    const body: ChatRequestBody = {
      question,
      ...(sessionId.value ? { sessionId: sessionId.value } : {}),
    }
    const response = await fetch(`${props.apiBaseUrl}/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
      signal: controller.signal,
    })
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }
    if (!response.body) {
      throw new Error('当前浏览器不支持流式响应')
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    let done = false

    while (!done) {
      const { value, done: readerDone } = await reader.read()
      if (readerDone) {
        break
      }
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split(/\r?\n/)
      buffer = lines.pop() ?? ''

      for (const rawLine of lines) {
        const line = rawLine.trim()
        if (!line.startsWith('data:')) {
          continue
        }
        const data = line.slice(5).trim()
        if (data === '[DONE]') {
          done = true
          break
        }
        const payload = parseChatEventPayload(data)
        if (payload) {
          applyPayload(payload)
        }
      }
    }
  } catch (error: unknown) {
    handleChatError(error)
  } finally {
    if (abortController === controller) {
      abortController = null
    }
    isStreaming.value = false
    await scrollToBottom()
  }
}

async function handleSend(): Promise<void> {
  const question = draft.value.trim()
  if (!question || isStreaming.value) {
    return
  }
  messages.value.push({ id: createMessageId(), role: 'user', content: question })
  messages.value.push({ id: createMessageId(), role: 'assistant', content: '' })
  draft.value = ''
  await scrollToBottom()
  await sendChat(question)
}

onBeforeUnmount(() => {
  abortController?.abort()
})
</script>

<style scoped>
.chat-page {
  display: flex;
  flex-direction: column;
  min-height: 100dvh;
  background: #f4f6f7;
  color: #1f2933;
  font-family:
    'PingFang SC',
    'Microsoft YaHei',
    -apple-system,
    BlinkMacSystemFont,
    'Segoe UI',
    sans-serif;
}

.chat-page__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  background: #ffffff;
  border-bottom: 1px solid #e3e8ea;
}

.chat-page__identity {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.chat-page__title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.chat-page__subtitle {
  margin: 0;
  font-size: 12px;
  color: #8a94a0;
}

.chat-page__status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #5b6672;
}

.chat-page__status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #22a06b;
}

.chat-page__status--active .chat-page__status-dot {
  background: #e6a23c;
  animation: chat-pulse 1.2s ease-in-out infinite;
}

.chat-page__messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.chat-message {
  display: flex;
}

.chat-message--user {
  justify-content: flex-end;
}

.chat-message--assistant {
  justify-content: flex-start;
}

.chat-message__bubble {
  max-width: min(78%, 680px);
  padding: 11px 14px;
  border-radius: 8px;
  font-size: 15px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.chat-message--user .chat-message__bubble {
  background: #128c7e;
  color: #ffffff;
}

.chat-message--assistant .chat-message__bubble {
  background: #ffffff;
  border: 1px solid #e3e8ea;
}

.chat-message__bubble--typing {
  display: flex;
  align-items: center;
  gap: 2px;
  color: #8a94a0;
}

.chat-message__ellipsis {
  display: inline-flex;
  gap: 3px;
}

.chat-message__ellipsis::before,
.chat-message__ellipsis::after {
  content: '';
}

.chat-message__ellipsis::before,
.chat-message__ellipsis::after,
.chat-message__ellipsis {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #8a94a0;
  animation: chat-blink 1.2s infinite;
}

.chat-message__ellipsis::after {
  animation-delay: 0.2s;
}

.chat-page__composer {
  padding: 14px 20px 18px;
  background: #ffffff;
  border-top: 1px solid #e3e8ea;
}

.chat-composer {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 900px;
  margin: 0 auto;
}

.chat-composer__input {
  width: 100%;
  resize: none;
  box-sizing: border-box;
  padding: 12px 14px;
  border: 1px solid #d5dce1;
  border-radius: 8px;
  font-family: inherit;
  font-size: 15px;
  line-height: 1.6;
  color: #1f2933;
  outline: none;
  transition: border-color 0.2s ease;
}

.chat-composer__input:focus {
  border-color: #128c7e;
}

.chat-composer__actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.chat-composer__hint {
  font-size: 12px;
  color: #8a94a0;
}

.chat-composer__button {
  min-width: 88px;
  height: 38px;
  border: none;
  border-radius: 8px;
  background: #128c7e;
  color: #ffffff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s ease, opacity 0.2s ease;
}

.chat-composer__button:hover:not(:disabled) {
  background: #0f7a6e;
}

.chat-composer__button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

@media (max-width: 640px) {
  .chat-page__messages {
    padding: 14px;
  }

  .chat-message__bubble {
    max-width: 88%;
  }

  .chat-page__composer {
    padding: 12px 14px 14px;
  }

  .chat-composer__hint {
    display: none;
  }
}

@keyframes chat-pulse {
  0%,
  100% {
    opacity: 1;
  }

  50% {
    opacity: 0.35;
  }
}

@keyframes chat-blink {
  0%,
  60%,
  100% {
    opacity: 0.3;
  }

  30% {
    opacity: 1;
  }
}
</style>
