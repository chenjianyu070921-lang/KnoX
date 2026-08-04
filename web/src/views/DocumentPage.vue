<template>
  <div class="doc-layout">
    <!-- ===== 顶部导航栏 ===== -->
    <header class="doc-navbar">
      <div class="doc-navbar-left">
        <button class="doc-navbar-back" v-if="viewMode === 'detail'" @click="backToList">
          ← 文档库
        </button>
        <h1 class="doc-navbar-title" v-else>📄 文档库</h1>
      </div>
      <div class="doc-navbar-right">
        <!-- 搜索框（列表页） -->
        <div class="doc-search" v-if="viewMode === 'list'">
          <input
            v-model="searchKeyword"
            class="doc-search-input"
            placeholder="搜索文档标题..."
            @input="onSearch"
          />
        </div>
        <!-- 详情页工具栏 -->
        <template v-if="viewMode === 'detail'">
          <button class="doc-btn doc-btn-outline" @click="toggleToc">
            {{ showToc ? '隐藏目录' : '显示目录' }}
          </button>
          <button class="doc-btn doc-btn-outline" @click="searchInDoc">
            搜索文中
          </button>
        </template>
      </div>
    </header>

    <!-- ===== 主体区域 ===== -->
    <div class="doc-body">
      <!-- 侧边栏 TOC（详情页） -->
      <aside
        class="doc-sidebar"
        :class="{ 'doc-sidebar--hidden': !showToc }"
        v-if="viewMode === 'detail'"
      >
        <div class="doc-toc-title">📑 目录</div>
        <nav class="doc-toc-nav" ref="tocNav">
          <a
            v-for="h in tocHeadings"
            :key="h.id"
            :href="`#${h.id}`"
            class="doc-toc-item"
            :class="{
              'doc-toc-item--h2': h.level === 2,
              'doc-toc-item--h3': h.level === 3,
              'doc-toc-item--active': activeHeading === h.id,
            }"
            @click.prevent="scrollToHeading(h.id)"
          >
            {{ h.text }}
          </a>
        </nav>
        <!-- 文档元信息 -->
        <div class="doc-meta">
          <div class="doc-meta-item">
            <span class="doc-meta-label">类型</span>
            <span class="doc-meta-badge">{{ currentDoc?.docType || '-' }}</span>
          </div>
          <div class="doc-meta-item">
            <span class="doc-meta-label">版本</span>
            <span class="doc-meta-badge">v{{ currentDoc?.version || 1 }}</span>
          </div>
          <div class="doc-meta-item">
            <span class="doc-meta-label">创建</span>
            <span class="doc-meta-text">{{ currentDoc?.createdAt || '-' }}</span>
          </div>
        </div>
      </aside>

      <!-- 主内容区 -->
      <main class="doc-main" ref="mainContent">
        <!-- ★ 列表视图 ★ -->
        <section v-if="viewMode === 'list'" class="doc-list-section">
          <!-- 上传区域 -->
          <div class="doc-upload-zone" @click="triggerUpload">
            <input
              ref="fileInput"
              type="file"
              class="doc-upload-input"
              accept=".txt,.md,.html,.htm,.json,.csv,.yaml,.yml,.xml,.py,.go,.java,.ts,.js"
              @change="onFileSelected"
            />
            <span class="doc-upload-icon">☁️</span>
            <span class="doc-upload-text">点击上传文档 / 拖拽到此处</span>
            <span class="doc-upload-hint">支持 txt、md、html、代码文件等</span>
            <span v-if="uploading" class="doc-upload-progress">上传中...</span>
          </div>

          <!-- 统计栏 -->
          <div class="doc-toolbar">
            <span class="doc-toolbar-count">共 {{ totalDocs }} 篇文档</span>
          </div>

          <!-- 文档卡片网格 -->
          <div class="doc-grid" v-if="docList.length > 0">
            <article
              v-for="doc in docList"
              :key="doc.docId"
              class="doc-card"
              @click="openDoc(doc)"
            >
              <div class="doc-card-icon">{{ iconForType(doc.docType) }}</div>
              <div class="doc-card-body">
                <h3 class="doc-card-title">{{ doc.title || '未命名文档' }}</h3>
                <p class="doc-card-type">{{ doc.docType }}</p>
                <p class="doc-card-date">{{ doc.createdAt }}</p>
              </div>
            </article>
          </div>

          <!-- 空状态 -->
          <div class="doc-empty" v-else>
            <div class="doc-empty-icon">📭</div>
            <p>还没有上传任何文档</p>
            <p class="doc-empty-hint">点击上方区域开始上传</p>
          </div>

          <!-- 分页 -->
          <div class="doc-pagination" v-if="totalPages > 1">
            <button
              class="doc-page-btn"
              :disabled="currentPage <= 1"
              @click="goPage(currentPage - 1)"
            >
              «
            </button>
            <span
              v-for="p in visiblePages"
              :key="p"
              class="doc-page-num"
              :class="{ active: p === currentPage }"
              @click="goPage(p)"
            >
              {{ p }}
            </span>
            <button
              class="doc-page-btn"
              :disabled="currentPage >= totalPages"
              @click="goPage(currentPage + 1)"
            >
              »
            </button>
          </div>
        </section>

        <!-- ★ 详情视图 ★ -->
        <section v-else-if="viewMode === 'detail'" class="doc-detail-section">
          <!-- 文档标题栏 -->
          <div class="doc-detail-header">
            <h1 class="doc-detail-title">{{ currentDoc?.title || '未命名文档' }}</h1>
            <div class="doc-detail-metarow">
              <span class="doc-detail-meta">{{ currentDoc?.docType }}</span>
              <span class="doc-detail-meta">v{{ currentDoc?.version || 1 }}</span>
              <span class="doc-detail-meta">{{ currentDoc?.createdAt }}</span>
            </div>
          </div>

          <!-- 渲染后的 Markdown 内容 -->
          <div
            class="doc-render"
            ref="docRender"
            v-html="renderedContent"
            @click="onContentClick"
          ></div>

          <!-- 底部备注区 -->
          <footer class="doc-footer">
            <div class="doc-footer-note">
              <span class="doc-footer-icon">📝</span>
              <span>文档 ID: {{ currentDoc?.docId }}</span>
            </div>
          </footer>
        </section>
      </main>
    </div>

    <!-- ===== 浮动工具：返回顶部 ===== -->
    <button
      class="doc-backtop"
      :class="{ 'doc-backtop--visible': showBackTop }"
      @click="scrollToTop"
      title="返回顶部"
    >
      ↑
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { marked } from 'marked'
import type { DocItem, DocListResp, DocDetailResp } from '../types'

// ==================== 状态 ====================

const viewMode = ref<'list' | 'detail'>('list')
const showToc = ref(true)
const showBackTop = ref(false)
const uploading = ref(false)

// 列表
const docList = ref<DocItem[]>([])
const totalDocs = ref(0)
const currentPage = ref(1)
const pageSize = 20
const searchKeyword = ref('')

// 详情
const currentDoc = ref<DocDetailResp | null>(null)
const renderedContent = ref('')
const tocHeadings = ref<{ id: string; text: string; level: number }[]>([])
const activeHeading = ref('')

// refs
const fileInput = ref<HTMLInputElement>()
const mainContent = ref<HTMLElement>()
const docRender = ref<HTMLElement>()
const tocNav = ref<HTMLElement>()
const BASE = import.meta.env.DEV ? 'http://localhost:8080' : ''

// ==================== 计算属性 ====================

const totalPages = computed(() => Math.max(1, Math.ceil(totalDocs.value / pageSize)))
const visiblePages = computed(() => {
  const pages: number[] = []
  const start = Math.max(1, currentPage.value - 2)
  const end = Math.min(totalPages.value, currentPage.value + 2)
  for (let i = start; i <= end; i++) pages.push(i)
  return pages
})

// ==================== 文档列表 ====================

async function fetchDocs(page: number = 1) {
  const params = new URLSearchParams({ page: String(page), size: String(pageSize) })
  if (searchKeyword.value) params.set('keyword', searchKeyword.value)

  try {
    const res = await fetch(`${BASE}/api/v1/docs?${params}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data: DocListResp = await res.json()
    docList.value = data.items
    totalDocs.value = data.total
    currentPage.value = data.page
  } catch (e) {
    console.error('[DocPage] fetchDocs failed:', e)
  }
}

function onSearch() {
  currentPage.value = 1
  fetchDocs(1)
}

function goPage(p: number) {
  if (p < 1 || p > totalPages.value) return
  fetchDocs(p)
}

// ==================== 文档详情 ====================

async function openDoc(doc: DocItem) {
  try {
    const res = await fetch(`${BASE}/api/v1/docs/${encodeURIComponent(doc.docId)}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data: DocDetailResp = await res.json()
    currentDoc.value = data
    renderMarkdown(data.content || '')
    viewMode.value = 'detail'
    showToc.value = true
    await nextTick()
    scrollToTop()
    extractHeadings()
  } catch (e) {
    console.error('[DocPage] openDoc failed:', e)
  }
}

function backToList() {
  viewMode.value = 'list'
  currentDoc.value = null
  renderedContent.value = ''
  tocHeadings.value = []
}

// ==================== Markdown 渲染 ====================

let headingIdCounter = 0

function renderMarkdown(md: string) {
  headingIdCounter = 0
  // marked配置：给标题加锚点ID，美化代码块/表格/引用
  const renderer = new marked.Renderer()

  renderer.heading = function ({ text, depth }: { text: string; depth: number }) {
    const id = slugify(text)
    return `<h${depth} id="${id}" class="doc-heading doc-h${depth}">${text}</h${depth}>`
  }

  renderer.code = function ({ text, lang }: { text: string; lang?: string }) {
    const escaped = escapeHtml(text)
    const langLabel = lang ? `<span class="doc-code-lang">${lang}</span>` : ''
    return `<div class="doc-code-block"><div class="doc-code-header">${langLabel}<button class="doc-code-copy" data-code="${encodeURIComponent(text)}">复制</button></div><pre><code class="language-${lang || ''}">${escaped}</code></pre></div>`
  }

  renderer.blockquote = function ({ tokens }: { tokens: any[] }) {
    const text = marked.parser(tokens)
    const lower = text.toLowerCase()
    let cls = 'doc-blockquote'
    let icon = ''
    if (lower.includes('warning') || lower.includes('警告') || lower.includes('⚠')) {
      cls += ' doc-blockquote--warn'
      icon = '<span class="doc-blockquote-icon">⚠️</span>'
    } else if (lower.includes('note') || lower.includes('备注') || lower.includes('注意')) {
      cls += ' doc-blockquote--note'
      icon = '<span class="doc-blockquote-icon">📌</span>'
    } else if (lower.includes('tip') || lower.includes('提示') || lower.includes('重点') || lower.includes('important')) {
      cls += ' doc-blockquote--tip'
      icon = '<span class="doc-blockquote-icon">💡</span>'
    }
    return `<blockquote class="${cls}">${icon}${text}</blockquote>`
  }

  renderer.table = function (token: { header: Array<{ text: string }>; rows: Array<Array<{ text: string }>> }) {
    let html = '<div class="doc-table-wrap"><table class="doc-table"><thead><tr>'
    for (const h of token.header) html += `<th>${h.text}</th>`
    html += '</tr></thead><tbody>'
    for (const row of token.rows) {
      html += '<tr>'
      for (const cell of row) html += `<td>${cell.text}</td>`
      html += '</tr>'
    }
    html += '</tbody></table></div>'
    return html
  }

  renderer.list = function ({ items, ordered }: { items: any[]; ordered: boolean }) {
    const tag = ordered ? 'ol' : 'ul'
    let html = `<${tag} class="doc-list doc-list--${ordered ? 'ordered' : 'unordered'}">`
    for (const item of items) {
      html += `<li class="doc-list-item">${item.text || ''}</li>`
    }
    html += `</${tag}>`
    return html
  }

  // 配置marked
  marked.setOptions({
    breaks: true,
    gfm: true,
  })

  renderedContent.value = marked.parse(md, { renderer }) as string
}

function extractHeadings() {
  if (!docRender.value) return
  const headings = docRender.value.querySelectorAll('h1[id],h2[id],h3[id]')
  tocHeadings.value = Array.from(headings).map((h) => ({
    id: h.id,
    text: h.textContent || '',
    level: parseInt(h.tagName.charAt(1)),
  }))
}

// ==================== TOC / 滚动监听 ====================

function scrollToHeading(id: string) {
  const el = document.getElementById(id)
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  activeHeading.value = id
}

function onScroll() {
  // 返回顶部按钮
  const scrollTop = window.scrollY || document.documentElement.scrollTop
  showBackTop.value = scrollTop > 300

  // TOC 活跃标题高亮
  if (!docRender.value) return
  const headings = docRender.value.querySelectorAll('h1[id],h2[id],h3[id]')
  for (let i = headings.length - 1; i >= 0; i--) {
    const rect = headings[i].getBoundingClientRect()
    if (rect.top <= 120) {
      activeHeading.value = headings[i].id
      break
    }
    if (i === 0) activeHeading.value = ''
  }

  // TOC 滚动跟随
  if (tocNav.value) {
    const activeEl = tocNav.value.querySelector('.doc-toc-item--active')
    if (activeEl) {
      activeEl.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
  }
}

function toggleToc() {
  showToc.value = !showToc.value
}

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// ==================== 上传 ====================

function triggerUpload() {
  fileInput.value?.click()
}

async function onFileSelected(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', file)
    const res = await fetch(`${BASE}/api/v1/doc/upload`, {
      method: 'POST',
      body: formData,
    })
    if (!res.ok) {
      const errData = await res.json().catch(() => ({}))
      throw new Error((errData as any).message || `上传失败 (${res.status})`)
    }
    alert(`✅ 文档 "${file.name}" 上传成功！`)
    input.value = ''
    fetchDocs(1)
  } catch (err: any) {
    alert(`❌ 上传失败: ${err.message}`)
  } finally {
    uploading.value = false
  }
}

// ==================== 文中搜索（简单） ====================

function searchInDoc() {
  const keyword = prompt('输入搜索关键词：')
  if (!keyword || !docRender.value) return
  const html = docRender.value.innerHTML
  const regex = new RegExp(`(${escapeRegex(keyword)})`, 'gi')
  docRender.value.innerHTML = html.replace(regex, '<mark class="doc-highlight">$1</mark>')
  // 滚动到第一个高亮
  const mark = docRender.value.querySelector('.doc-highlight')
  if (mark) mark.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

// ==================== 内容点击（代码复制） ====================

function onContentClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (target.classList.contains('doc-code-copy')) {
    const code = decodeURIComponent(target.dataset.code || '')
    navigator.clipboard.writeText(code).then(() => {
      target.textContent = '已复制!'
      setTimeout(() => (target.textContent = '复制'), 1500)
    })
  }
}

// ==================== 工具函数 ====================

function slugify(text: string): string {
  headingIdCounter++
  const slug = text
    .replace(/[^\w\u4e00-\u9fff]/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
    .toLowerCase()
  return slug || `heading-${headingIdCounter}`
}

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function escapeRegex(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function iconForType(t: string): string {
  const map: Record<string, string> = {
    md: '📝',
    markdown: '📝',
    txt: '📃',
    html: '🌐',
    htm: '🌐',
    json: '📊',
    csv: '📈',
    yaml: '⚙️',
    yml: '⚙️',
    xml: '📋',
    py: '🐍',
    go: '🔵',
    java: '☕',
    ts: '🔷',
    js: '🟨',
  }
  return map[t.toLowerCase()] || '📄'
}

// ==================== 生命周期 ====================

onMounted(() => {
  fetchDocs(1)
  window.addEventListener('scroll', onScroll, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('scroll', onScroll)
})
</script>

<style scoped>
/* ================================================================
   CSS 变量 / 色彩体系
   ================================================================ */
:root {
  --doc-bg: #f5f6fa;
  --doc-white: #ffffff;
  --doc-text: #2c3e50;
  --doc-text-secondary: #7f8c9b;
  --doc-border: #e2e6ed;
  --doc-accent: #4a6cf7;
  --doc-accent-light: #eef1ff;
  --doc-success: #22c55e;
  --doc-warn: #f59e0b;
  --doc-danger: #ef4444;
  --doc-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04);
  --doc-shadow-lg: 0 4px 24px rgba(0,0,0,0.08);
  --doc-radius: 10px;
  --doc-radius-sm: 6px;
  --doc-font: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif;
}

/* ================================================================
   全局容器
   ================================================================ */
.doc-layout {
  font-family: var(--doc-font);
  background: var(--doc-bg);
  min-height: 100vh;
  color: var(--doc-text);
  line-height: 1.6;
  display: flex;
  flex-direction: column;
}

/* ================================================================
   顶部导航
   ================================================================ */
.doc-navbar {
  position: sticky;
  top: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 24px;
  background: var(--doc-white);
  border-bottom: 1px solid var(--doc-border);
  box-shadow: 0 1px 2px rgba(0,0,0,0.03);
}
.doc-navbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.doc-navbar-back {
  background: none;
  border: none;
  color: var(--doc-accent);
  font-size: 14px;
  cursor: pointer;
  padding: 6px 12px;
  border-radius: var(--doc-radius-sm);
  transition: background .15s;
}
.doc-navbar-back:hover {
  background: var(--doc-accent-light);
}
.doc-navbar-title {
  font-size: 18px;
  font-weight: 700;
  margin: 0;
  color: var(--doc-text);
}
.doc-navbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

/* 搜索 */
.doc-search {
  position: relative;
}
.doc-search-input {
  width: 260px;
  padding: 8px 14px 8px 36px;
  border: 1px solid var(--doc-border);
  border-radius: 20px;
  font-size: 14px;
  outline: none;
  background: var(--doc-bg);
  transition: all .2s;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%237f8c9b' stroke-width='2'%3E%3Ccircle cx='11' cy='11' r='8'%3E%3C/circle%3E%3Cpath d='m21 21-4.35-4.35'%3E%3C/path%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: 12px center;
}
.doc-search-input:focus {
  border-color: var(--doc-accent);
  background-color: var(--doc-white);
  box-shadow: 0 0 0 3px var(--doc-accent-light);
}

/* 通用小按钮 */
.doc-btn {
  font-size: 13px;
  padding: 6px 14px;
  border-radius: var(--doc-radius-sm);
  cursor: pointer;
  transition: all .15s;
  border: 1px solid var(--doc-border);
  background: var(--doc-white);
  color: var(--doc-text);
}
.doc-btn:hover {
  border-color: var(--doc-accent);
  color: var(--doc-accent);
}
.doc-btn-outline {}

/* ================================================================
   主体：侧边栏 + 内容
   ================================================================ */
.doc-body {
  display: flex;
  flex: 1;
  max-width: 1280px;
  width: 100%;
  margin: 0 auto;
  padding: 24px 24px 60px;
  gap: 32px;
}

/* ---------- 侧边栏 ---------- */
.doc-sidebar {
  position: sticky;
  top: 80px;
  width: 240px;
  flex-shrink: 0;
  align-self: flex-start;
  max-height: calc(100vh - 120px);
  display: flex;
  flex-direction: column;
  background: var(--doc-white);
  border-radius: var(--doc-radius);
  box-shadow: var(--doc-shadow);
  overflow: hidden;
  transition: all .3s;
}
.doc-sidebar--hidden {
  width: 0;
  min-width: 0;
  opacity: 0;
  padding: 0;
  margin: 0;
  overflow: hidden;
}
.doc-toc-title {
  padding: 16px 16px 10px;
  font-size: 13px;
  font-weight: 700;
  color: var(--doc-text-secondary);
  text-transform: uppercase;
  letter-spacing: .5px;
  border-bottom: 1px solid var(--doc-border);
}
.doc-toc-nav {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
}
.doc-toc-item {
  display: block;
  padding: 6px 10px;
  font-size: 13px;
  color: var(--doc-text-secondary);
  text-decoration: none;
  border-left: 2px solid transparent;
  border-radius: 0 var(--doc-radius-sm) var(--doc-radius-sm) 0;
  transition: all .15s;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.doc-toc-item:hover {
  color: var(--doc-accent);
  background: var(--doc-accent-light);
}
.doc-toc-item--active {
  color: var(--doc-accent);
  border-left-color: var(--doc-accent);
  font-weight: 600;
  background: var(--doc-accent-light);
}
.doc-toc-item--h3 {
  padding-left: 24px;
  font-size: 12px;
}

/* 侧边栏 - 文档元信息 */
.doc-meta {
  border-top: 1px solid var(--doc-border);
  padding: 12px 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.doc-meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}
.doc-meta-label {
  color: var(--doc-text-secondary);
}
.doc-meta-badge {
  background: var(--doc-accent-light);
  color: var(--doc-accent);
  padding: 1px 8px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
}
.doc-meta-text {
  color: var(--doc-text);
}

/* ---------- 主内容 ---------- */
.doc-main {
  flex: 1;
  min-width: 0;
}

/* ================================================================
   列表视图
   ================================================================ */
.doc-list-section {}

/* 上传区域 */
.doc-upload-zone {
  border: 2px dashed var(--doc-border);
  border-radius: var(--doc-radius);
  padding: 32px;
  text-align: center;
  cursor: pointer;
  transition: all .2s;
  background: var(--doc-white);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.doc-upload-zone:hover {
  border-color: var(--doc-accent);
  background: var(--doc-accent-light);
}
.doc-upload-input {
  display: none;
}
.doc-upload-icon {
  font-size: 36px;
}
.doc-upload-text {
  font-size: 15px;
  color: var(--doc-text);
  font-weight: 500;
}
.doc-upload-hint {
  font-size: 12px;
  color: var(--doc-text-secondary);
}
.doc-upload-progress {
  font-size: 13px;
  color: var(--doc-accent);
  font-weight: 600;
}

/* 工具栏 */
.doc-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 20px 0 16px;
}
.doc-toolbar-count {
  font-size: 13px;
  color: var(--doc-text-secondary);
}

/* 卡片网格 */
.doc-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}
.doc-card {
  background: var(--doc-white);
  border-radius: var(--doc-radius);
  box-shadow: var(--doc-shadow);
  padding: 20px;
  cursor: pointer;
  transition: all .2s;
  display: flex;
  gap: 14px;
  align-items: flex-start;
}
.doc-card:hover {
  box-shadow: var(--doc-shadow-lg);
  transform: translateY(-2px);
}
.doc-card-icon {
  font-size: 28px;
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--doc-bg);
  border-radius: var(--doc-radius-sm);
}
.doc-card-body {
  min-width: 0;
}
.doc-card-title {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 600;
  color: var(--doc-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.doc-card-type {
  margin: 0 0 2px;
  font-size: 12px;
  color: var(--doc-accent);
  font-weight: 500;
}
.doc-card-date {
  margin: 0;
  font-size: 11px;
  color: var(--doc-text-secondary);
}

/* 空状态 */
.doc-empty {
  text-align: center;
  padding: 80px 20px;
  color: var(--doc-text-secondary);
}
.doc-empty-icon {
  font-size: 48px;
  margin-bottom: 12px;
}
.doc-empty-hint {
  font-size: 13px;
  opacity: .7;
}

/* 分页 */
.doc-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-top: 28px;
}
.doc-page-btn {
  border: 1px solid var(--doc-border);
  background: var(--doc-white);
  padding: 6px 14px;
  border-radius: var(--doc-radius-sm);
  cursor: pointer;
  font-size: 14px;
  color: var(--doc-text);
  transition: all .15s;
}
.doc-page-btn:disabled {
  opacity: .4;
  cursor: not-allowed;
}
.doc-page-btn:not(:disabled):hover {
  border-color: var(--doc-accent);
  color: var(--doc-accent);
}
.doc-page-num {
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--doc-radius-sm);
  cursor: pointer;
  font-size: 13px;
  color: var(--doc-text-secondary);
  transition: all .15s;
}
.doc-page-num:hover {
  background: var(--doc-accent-light);
  color: var(--doc-accent);
}
.doc-page-num.active {
  background: var(--doc-accent);
  color: #fff;
  font-weight: 600;
}

/* ================================================================
   详情视图
   ================================================================ */
.doc-detail-section {
  background: var(--doc-white);
  border-radius: var(--doc-radius);
  box-shadow: var(--doc-shadow);
  overflow: hidden;
}

/* 标题栏 */
.doc-detail-header {
  padding: 28px 36px 20px;
  border-bottom: 1px solid var(--doc-border);
  background: linear-gradient(135deg, #f8faff 0%, #fff 100%);
}
.doc-detail-title {
  margin: 0 0 10px;
  font-size: 26px;
  font-weight: 800;
  color: var(--doc-text);
  line-height: 1.3;
}
.doc-detail-metarow {
  display: flex;
  gap: 16px;
}
.doc-detail-meta {
  font-size: 12px;
  color: var(--doc-text-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
}
.doc-detail-meta::after {
  content: '·';
}
.doc-detail-meta:last-child::after {
  display: none;
}

/* 渲染内容区 */
.doc-render {
  padding: 36px 36px 48px;
}

/* ================================================================
   Markdown 渲染样式（逐元素美化）
   ================================================================ */
.doc-render :deep(h1.doc-heading),
.doc-render :deep(h2.doc-heading),
.doc-render :deep(h3.doc-heading),
.doc-render :deep(h4.doc-heading) {
  color: var(--doc-text);
  margin: 1.8em 0 .6em;
  padding-bottom: .3em;
  scroll-margin-top: 80px;
}
.doc-render :deep(h1.doc-heading) {
  font-size: 1.75em;
  font-weight: 800;
  border-bottom: 2px solid var(--doc-accent);
  padding-bottom: .35em;
}
.doc-render :deep(h2.doc-heading) {
  font-size: 1.35em;
  font-weight: 700;
  border-bottom: 1px solid var(--doc-border);
}
.doc-render :deep(h3.doc-heading) {
  font-size: 1.15em;
  font-weight: 600;
}
.doc-render :deep(h4.doc-heading) {
  font-size: 1.05em;
  font-weight: 600;
}
.doc-render :deep(p) {
  margin: .8em 0;
  font-size: 15px;
  line-height: 1.8;
  color: #374151;
}

/* 内联代码 */
.doc-render :deep(code:not(pre code)) {
  background: #f1f5f9;
  color: #e11d48;
  padding: 2px 7px;
  border-radius: 4px;
  font-size: .9em;
  font-family: 'Fira Code', 'Cascadia Code', 'Source Code Pro', Consolas, monospace;
}

/* 代码块 */
.doc-code-block {
  margin: 1.2em 0;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid #e2e8f0;
  background: #1e293b;
}
.doc-code-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 14px;
  background: #334155;
  font-size: 12px;
}
.doc-code-lang {
  color: #94a3b8;
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: .5px;
}
.doc-code-copy {
  background: rgba(255,255,255,.1);
  border: none;
  color: #94a3b8;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 11px;
  cursor: pointer;
  transition: all .15s;
}
.doc-code-copy:hover {
  background: rgba(255,255,255,.2);
  color: #fff;
}
.doc-code-block :deep(pre) {
  margin: 0;
  padding: 16px 18px;
  overflow-x: auto;
  background: #1e293b;
}
.doc-code-block :deep(code) {
  color: #e2e8f0;
  font-size: 13.5px;
  line-height: 1.7;
  font-family: 'Fira Code', 'Cascadia Code', 'Source Code Pro', Consolas, monospace;
  background: none !important;
  padding: 0 !important;
}

/* 引用块 */
.doc-render :deep(blockquote) {
  margin: 1.2em 0;
  padding: 14px 18px 14px 44px;
  border-radius: 8px;
  position: relative;
  font-size: 14px;
  line-height: 1.7;
  color: #475569;
}
.doc-render :deep(.doc-blockquote) {
  background: #f8fafc;
  border-left: 4px solid #cbd5e1;
}
.doc-render :deep(.doc-blockquote--warn) {
  background: #fffbeb;
  border-left-color: var(--doc-warn);
}
.doc-render :deep(.doc-blockquote--note) {
  background: #f0f9ff;
  border-left-color: var(--doc-accent);
}
.doc-render :deep(.doc-blockquote--tip) {
  background: #f0fdf4;
  border-left-color: var(--doc-success);
}
.doc-blockquote-icon {
  position: absolute;
  left: 14px;
  top: 14px;
  font-size: 16px;
}

/* 表格 */
.doc-table-wrap {
  margin: 1em 0;
  overflow-x: auto;
  border-radius: 8px;
  border: 1px solid var(--doc-border);
}
.doc-render :deep(.doc-table) {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}
.doc-render :deep(.doc-table thead) {
  background: #f8fafc;
}
.doc-render :deep(.doc-table th) {
  padding: 10px 14px;
  text-align: left;
  font-weight: 600;
  color: var(--doc-text);
  border-bottom: 2px solid var(--doc-border);
  font-size: 13px;
  white-space: nowrap;
}
.doc-render :deep(.doc-table td) {
  padding: 9px 14px;
  border-bottom: 1px solid var(--doc-border);
  color: #475569;
}
.doc-render :deep(.doc-table tbody tr:hover) {
  background: #f8fafc;
}

/* 列表 */
.doc-render :deep(.doc-list) {
  padding-left: 24px;
  margin: .8em 0;
}
.doc-render :deep(.doc-list-item) {
  margin: 4px 0;
  font-size: 15px;
  line-height: 1.8;
  color: #374151;
}
.doc-render :deep(.doc-list-item::marker) {
  color: var(--doc-accent);
}

/* 链接 */
.doc-render :deep(a) {
  color: var(--doc-accent);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: border-color .15s;
}
.doc-render :deep(a:hover) {
  border-bottom-color: var(--doc-accent);
}

/* 图片 */
.doc-render :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: .8em 0;
}

/* 分割线 */
.doc-render :deep(hr) {
  border: none;
  height: 1px;
  background: var(--doc-border);
  margin: 2em 0;
}

/* 高亮 */
.doc-render :deep(mark.doc-highlight) {
  background: #fef08a;
  padding: 1px 3px;
  border-radius: 3px;
}

/* 首段（摘要样式） */
.doc-render :deep(p:first-child) {
  font-size: 16px;
  color: #475569;
}

/* ================================================================
   底部备注
   ================================================================ */
.doc-footer {
  margin-top: 40px;
  padding: 20px 36px;
  border-top: 1px solid var(--doc-border);
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
}
.doc-footer-note {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--doc-text-secondary);
}
.doc-footer-icon {
  font-size: 14px;
}

/* ================================================================
   返回顶部按钮
   ================================================================ */
.doc-backtop {
  position: fixed;
  bottom: 32px;
  right: 32px;
  width: 42px;
  height: 42px;
  border-radius: 50%;
  background: var(--doc-accent);
  color: #fff;
  border: none;
  cursor: pointer;
  font-size: 18px;
  font-weight: 700;
  box-shadow: 0 4px 12px rgba(74,108,247,0.35);
  opacity: 0;
  transform: translateY(10px);
  pointer-events: none;
  transition: all .3s;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}
.doc-backtop--visible {
  opacity: 1;
  transform: translateY(0);
  pointer-events: auto;
}
.doc-backtop:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(74,108,247,0.45);
}

/* ================================================================
   响应式
   ================================================================ */
@media (max-width: 768px) {
  .doc-body {
    flex-direction: column;
    padding: 16px 12px 60px;
    gap: 16px;
  }
  .doc-sidebar {
    position: static;
    width: 100%;
    max-height: none;
  }
  .doc-grid {
    grid-template-columns: 1fr;
  }
  .doc-render {
    padding: 20px 18px 32px;
  }
  .doc-detail-header {
    padding: 20px 18px 16px;
  }
  .doc-navbar {
    padding: 0 12px;
  }
  .doc-search-input {
    width: 180px;
  }
}
</style>
