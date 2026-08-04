<template>
  <div class="dash">
    <!-- ===== 顶部 ===== -->
    <header class="dash-header">
      <div class="dash-header-left">
        <h1 class="dash-title">数据大盘</h1>
        <span v-if="usingMock" class="dash-badge">演示数据</span>
        <span v-else class="dash-badge dash-badge--live">实时</span>
      </div>
      <div class="dash-header-right">
        <span class="dash-updated">更新于 {{ updatedAt }}</span>
        <button class="dash-refresh" @click="fetchData" :class="{ spinning: loading }" title="刷新">
          ↻
        </button>
      </div>
    </header>

    <!-- ===== 四张概览卡片 ===== -->
    <section class="dash-cards">
      <div class="dash-card dash-card--chat" @mouseenter="hoverCard='chat'" @mouseleave="hoverCard=''">
        <div class="dash-card-icon">💬</div>
        <div class="dash-card-body">
          <div class="dash-card-val">{{ fmtNum(data.chatTotal) }}</div>
          <div class="dash-card-label">对话次数</div>
          <div class="dash-card-sub">平均 {{ fmtMs(data.chatAvgMs) }} · <span class="dash-trend up">↑ 12.4%</span></div>
        </div>
        <div class="dash-card-spark">
          <svg viewBox="0 0 80 30" preserveAspectRatio="none">
            <polyline :points="sparkPoints('chat')" fill="none" stroke="#4a6cf7" stroke-width="1.5" />
          </svg>
        </div>
      </div>

      <div class="dash-card dash-card--search" @mouseenter="hoverCard='search'" @mouseleave="hoverCard=''">
        <div class="dash-card-icon">🔍</div>
        <div class="dash-card-body">
          <div class="dash-card-val">{{ fmtNum(data.searchTotal) }}</div>
          <div class="dash-card-label">搜索次数</div>
          <div class="dash-card-sub">平均 {{ fmtMs(data.searchAvgMs) }} · <span class="dash-trend up">↑ 8.1%</span></div>
        </div>
        <div class="dash-card-spark">
          <svg viewBox="0 0 80 30" preserveAspectRatio="none">
            <polyline :points="sparkPoints('search')" fill="none" stroke="#f59e0b" stroke-width="1.5" />
          </svg>
        </div>
      </div>

      <div class="dash-card dash-card--upload" @mouseenter="hoverCard='upload'" @mouseleave="hoverCard=''">
        <div class="dash-card-icon">☁️</div>
        <div class="dash-card-body">
          <div class="dash-card-val">{{ fmtNum(data.uploadTotal) }}</div>
          <div class="dash-card-label">上传次数</div>
          <div class="dash-card-sub">平均 {{ fmtMs(data.uploadAvgMs) }} · <span class="dash-trend down">↓ 3.2%</span></div>
        </div>
        <div class="dash-card-spark">
          <svg viewBox="0 0 80 30" preserveAspectRatio="none">
            <polyline :points="sparkPoints('upload')" fill="none" stroke="#22c55e" stroke-width="1.5" />
          </svg>
        </div>
      </div>

      <div class="dash-card" :class="errColor" @mouseenter="hoverCard='err'" @mouseleave="hoverCard=''">
        <div class="dash-card-icon">{{ errIcon }}</div>
        <div class="dash-card-body">
          <div class="dash-card-val">{{ fmtPct(data.errorRate) }}</div>
          <div class="dash-card-label">错误率</div>
          <div class="dash-card-sub">
            P50 {{ fmtMs(data.p50Ms) }} · P95 {{ fmtMs(data.p95Ms) }} · P99 {{ fmtMs(data.p99Ms) }}
          </div>
        </div>
      </div>
    </section>

    <!-- ===== 图表区 ===== -->
    <div class="dash-charts">
      <!-- 24h 请求量趋势（柱状图）-->
      <section class="dash-panel dash-panel--wide">
        <div class="dash-panel-head">
          <h3 class="dash-panel-title">24h 请求量趋势</h3>
          <div class="dash-panel-tabs">
            <button v-for="t in ['all','chat','search','upload']" :key="t"
              :class="{ active: barTab===t }" @click="barTab=t as any">{{ tabLabel(t) }}</button>
          </div>
        </div>
        <div class="dash-bars">
          <div
            v-for="(h, i) in data.hourly"
            :key="i"
            class="dash-bar-group"
            @mouseenter="hoverBar=i" @mouseleave="hoverBar=-1"
          >
            <div class="dash-bar-stack">
              <div v-if="barTab==='all' || barTab==='chat'"
                class="dash-bar dash-bar--chat"
                :style="{ height: barPct(h.chatCnt, maxHourly(barTab)), animationDelay: i*20+'ms' }"></div>
              <div v-if="barTab==='all' || barTab==='search'"
                class="dash-bar dash-bar--search"
                :style="{ height: barPct(h.searchCnt, maxHourly(barTab)), animationDelay: i*20+10+'ms' }"></div>
              <div v-if="barTab==='all' || barTab==='upload'"
                class="dash-bar dash-bar--upload"
                :style="{ height: barPct(h.uploadCnt, maxHourly(barTab)), animationDelay: i*20+20+'ms' }"></div>
            </div>
            <span class="dash-bar-time">{{ hourShort(h.hour) }}</span>
            <div v-if="hoverBar===i" class="dash-bar-tip">
              {{ hourLabel(h.hour) }}<br>
              💬 {{ h.chatCnt }} · 🔍 {{ h.searchCnt }} · ☁️ {{ h.uploadCnt }}
            </div>
          </div>
        </div>
        <div class="dash-legend">
          <span class="dash-legend-dot dash-legend-dot--chat"></span> 对话
          <span class="dash-legend-dot dash-legend-dot--search"></span> 搜索
          <span class="dash-legend-dot dash-legend-dot--upload"></span> 上传
        </div>
      </section>

      <!-- 事件分布（环形图）-->
      <section class="dash-panel">
        <h3 class="dash-panel-title">事件分布</h3>
        <div class="dash-donut-wrap">
          <div class="dash-donut-container">
            <svg viewBox="0 0 160 160" class="dash-donut">
              <circle
                v-for="(seg, i) in donutSegs"
                :key="seg.type"
                cx="80" cy="80" r="60"
                fill="none"
                stroke-width="22"
                :stroke="seg.color"
                :stroke-dasharray="seg.dash + ' ' + (totalRing - seg.dash)"
                :stroke-dashoffset="-seg.offsetDeg"
                :style="{
                  transform: 'rotate(' + (-90 + seg.rotated) + 'deg)',
                  transformOrigin: '50% 50%',
                  opacity: hoverDonut === -1 || hoverDonut === i ? 1 : 0.35
                }"
                @mouseenter="hoverDonut=i" @mouseleave="hoverDonut=-1"
              />
            </svg>
            <div class="dash-donut-center">
              <div class="dash-donut-total">{{ totalEvents }}</div>
              <div class="dash-donut-label">总事件</div>
            </div>
          </div>
          <div class="dash-legend dash-legend--col">
            <div v-for="(seg, i) in donutSegs" :key="seg.type" class="dash-legend-row"
              @mouseenter="hoverDonut=i" @mouseleave="hoverDonut=-1">
              <span class="dash-legend-dot" :style="{ background: seg.color }"></span>
              <span class="dash-legend-name">{{ seg.type }}</span>
              <span class="dash-legend-val">{{ seg.cnt }} · {{ seg.pct }}%</span>
            </div>
          </div>
        </div>
      </section>

      <!-- 性能分位 -->
      <section class="dash-panel">
        <h3 class="dash-panel-title">延迟分位</h3>
        <div class="dash-percentile">
          <div v-for="(p, i) in percentiles" :key="p.label" class="dash-percentile-row"
            :class="{ active: hoverPctl === i }" @mouseenter="hoverPctl=i" @mouseleave="hoverPctl=-1">
            <div class="dash-percentile-label">{{ p.label }}</div>
            <div class="dash-percentile-bar">
              <div class="dash-percentile-fill" :style="{
                width: p.pct + '%',
                background: p.color,
                animationDelay: i * 100 + 'ms'
              }"></div>
            </div>
            <div class="dash-percentile-val">{{ fmtMs(p.val) }}</div>
          </div>
          <div class="dash-percentile-hint">
            💡 P99 越低越好；理想情况 P99 &lt; 3×P50
          </div>
        </div>
      </section>

      <!-- 7天成功率（SVG 折线图）-->
      <section class="dash-panel dash-panel--wide">
        <div class="dash-panel-head">
          <h3 class="dash-panel-title">7天成功率趋势</h3>
          <div class="dash-panel-meta">
            <span class="dash-stat-pill dash-stat-pill--up">↑ {{ trendDelta }}% vs 上周</span>
          </div>
        </div>
        <svg class="dash-line-chart" viewBox="0 0 600 220" @mouseleave="hoverLine=-1">
          <!-- 网格 -->
          <g v-for="y in 4" :key="'g'+y">
            <line x1="40" :y1="20+y*45" x2="580" :y2="20+y*45" stroke="#e2e8f0" stroke-dasharray="2,3" />
            <text x="36" :y="24+y*45" text-anchor="end" font-size="10" fill="#94a3b8">{{ 120-y*25 }}%</text>
          </g>
          <!-- 100% 参考线 -->
          <line x1="40" y1="20" x2="580" y2="20" stroke="#22c55e" stroke-dasharray="4,4" opacity="0.4" />
          <text x="585" y="23" font-size="9" fill="#22c55e">100%</text>
          <!-- 面积 -->
          <polygon v-if="successArea" :points="successArea" fill="url(#successGrad)" />
          <!-- 折线 -->
          <polyline v-if="successLine" :points="successLine" fill="none"
            stroke="#22c55e" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
          <!-- 数据点 + tooltip -->
          <g v-for="(pt, i) in successDots" :key="'d'+i" class="dash-line-pt"
            @mouseenter="hoverLine=i">
            <circle :cx="pt.x" :cy="pt.y" r="4" fill="#fff" stroke="#22c55e" stroke-width="2.5" />
            <circle :cx="pt.x" :cy="pt.y" r="12" fill="transparent" />
            <g v-if="hoverLine===i" :transform="`translate(${pt.x}, ${pt.y-30})`">
              <rect x="-32" y="-22" width="64" height="28" rx="4" fill="#1e293b" opacity="0.92" />
              <text x="0" y="-10" text-anchor="middle" font-size="10" fill="#fff">
                {{ dateShort(data.successRate7d[i]?.date || '') }}
              </text>
              <text x="0" y="0" text-anchor="middle" font-size="11" fill="#22c55e" font-weight="700">
                {{ data.successRate7d[i]?.rate?.toFixed(1) }}%
              </text>
            </g>
          </g>
          <!-- X 轴 -->
          <text v-for="(pt, i) in successDots" :key="'l'+i"
            :x="pt.x" y="210" text-anchor="middle" font-size="10" fill="#94a3b8">
            {{ dateShort(data.successRate7d[i]?.date || '') }}
          </text>
          <!-- 渐变 -->
          <defs>
            <linearGradient id="successGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stop-color="#22c55e" stop-opacity="0.35" />
              <stop offset="100%" stop-color="#22c55e" stop-opacity="0" />
            </linearGradient>
          </defs>
        </svg>
      </section>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

// ============ 数据接口 ============

interface HourlyPoint {
  hour: string; chatCnt: number; searchCnt: number; uploadCnt: number; avgMs: number
}
interface DistItem { eventType: string; cnt: number }
interface SuccessRatePoint { date: string; total: number; successCnt: number; rate: number }

interface DashboardData {
  chatTotal: number; searchTotal: number; uploadTotal: number
  chatAvgMs: number; searchAvgMs: number; uploadAvgMs: number
  errorRate: number; p50Ms: number; p95Ms: number; p99Ms: number
  hourly: HourlyPoint[]; distribution: DistItem[]; successRate7d: SuccessRatePoint[]
}

// ============ 状态 ============

const data = ref<DashboardData>(emptyData())
const loading = ref(false)
const usingMock = ref(false)
const updatedAt = ref('--:--:--')
const hoverBar = ref(-1)
const hoverDonut = ref(-1)
const hoverLine = ref(-1)
const hoverPctl = ref(-1)
const hoverCard = ref('')
const barTab = ref<'all'|'chat'|'search'|'upload'>('all')
let timer: number | undefined

const REPORTER_BASE = import.meta.env.DEV ? 'http://localhost:8081' : ''

function emptyData(): DashboardData {
  return {
    chatTotal: 0, searchTotal: 0, uploadTotal: 0,
    chatAvgMs: 0, searchAvgMs: 0, uploadAvgMs: 0,
    errorRate: 0, p50Ms: 0, p95Ms: 0, p99Ms: 0,
    hourly: [], distribution: [], successRate7d: [],
  }
}

// ============ Mock 数据生成 ============

function rand(min: number, max: number) {
  return Math.random() * (max - min) + min
}
function randInt(min: number, max: number) {
  return Math.floor(rand(min, max + 1))
}
function pad(n: number) { return n < 10 ? '0' + n : '' + n }

function genMockHourly(): HourlyPoint[] {
  const now = new Date()
  const arr: HourlyPoint[] = []
  for (let i = 23; i >= 0; i--) {
    const d = new Date(now.getTime() - i * 3600 * 1000)
    // 模拟工作日流量模式：凌晨低，工作时间高，午饭略低
    const h = d.getHours()
    let wave = 0
    if (h >= 9 && h <= 12) wave = 1.0
    else if (h >= 14 && h <= 18) wave = 0.95
    else if (h >= 19 && h <= 22) wave = 0.7
    else if (h >= 7 && h <= 8) wave = 0.6
    else wave = 0.18

    const base = wave * 80 + rand(0, 20)
    arr.push({
      hour: `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(h)}:00:00`,
      chatCnt: Math.round(base * 0.55 + rand(0, 12)),
      searchCnt: Math.round(base * 0.32 + rand(0, 8)),
      uploadCnt: Math.round(base * 0.13 + rand(0, 4)),
      avgMs: Math.round(rand(280, 1200)),
    })
  }
  return arr
}

function genMockDistribution(): DistItem[] {
  return [
    { eventType: 'chat',   cnt: 1247 },
    { eventType: 'search', cnt: 856 },
    { eventType: 'upload', cnt: 134 },
    { eventType: 'rag',    cnt: 92 },
    { eventType: 'eval',   cnt: 28 },
  ]
}

function genMockSuccessRate7d(): SuccessRatePoint[] {
  const arr: SuccessRatePoint[] = []
  const now = new Date()
  for (let i = 6; i >= 0; i--) {
    const d = new Date(now.getTime() - i * 86400 * 1000)
    const total = randInt(1200, 2400)
    const rate = 95 + rand(0, 4.5)
    arr.push({
      date: `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}`,
      total,
      successCnt: Math.round(total * rate / 100),
      rate,
    })
  }
  return arr
}

function genMockData(): DashboardData {
  return {
    chatTotal: 1247, chatAvgMs: 1245,
    searchTotal: 856, searchAvgMs: 312,
    uploadTotal: 134, uploadAvgMs: 1823,
    errorRate: 1.83,
    p50Ms: 285, p95Ms: 1820, p99Ms: 4250,
    hourly: genMockHourly(),
    distribution: genMockDistribution(),
    successRate7d: genMockSuccessRate7d(),
  }
}

// ============ 计算属性 ============

const maxHourly = computed(() => (tab: string) => {
  let m = 1
  for (const h of data.value.hourly) {
    if (tab === 'chat') m = Math.max(m, h.chatCnt)
    else if (tab === 'search') m = Math.max(m, h.searchCnt)
    else if (tab === 'upload') m = Math.max(m, h.uploadCnt)
    else m = Math.max(m, h.chatCnt + h.searchCnt + h.uploadCnt)
  }
  return m
})

const totalEvents = computed(() =>
  data.value.distribution.reduce((s, d) => s + d.cnt, 0)
)

const totalRing = computed(() => 2 * Math.PI * 60)

const donutSegs = computed(() => {
  const colors: Record<string, string> = {
    chat: '#4a6cf7', search: '#f59e0b', upload: '#22c55e',
    rag: '#8b5cf6', eval: '#ec4899', agent: '#06b6d4',
  }
  const defCli = ['#8b5cf6', '#ec4899', '#06b6d4']
  let ci = 0
  const total = totalEvents.value || 1
  let rotated = 0
  return data.value.distribution.map((d) => {
    const pct = totalEvents.value > 0 ? (d.cnt / totalEvents.value) * 100 : 0
    const dash = (d.cnt / total) * totalRing.value
    const seg = {
      type: d.eventType,
      color: colors[d.eventType] || defCli[ci++ % defCli.length],
      dash,
      offsetDeg: 0,
      rotated,
      cnt: d.cnt,
      pct: Math.round(pct),
    }
    rotated += (d.cnt / total) * 360
    return seg
  })
})

const percentiles = computed(() => [
  { label: 'P50', val: data.value.p50Ms, pct: pctBar(data.value.p50Ms, data.value.p99Ms), color: '#4a6cf7' },
  { label: 'P95', val: data.value.p95Ms, pct: pctBar(data.value.p95Ms, data.value.p99Ms), color: '#f59e0b' },
  { label: 'P99', val: data.value.p99Ms, pct: pctBar(data.value.p99Ms, data.value.p99Ms), color: '#ef4444' },
  { label: 'Max', val: data.value.p99Ms * 1.3, pct: pctBar(data.value.p99Ms * 1.3, data.value.p99Ms), color: '#94a3b8' },
])

const trendDelta = computed(() => {
  const arr = data.value.successRate7d
  if (arr.length < 2) return '0.0'
  const first = arr.slice(0, Math.floor(arr.length / 2))
  const last = arr.slice(Math.floor(arr.length / 2))
  const avg = (xs: SuccessRatePoint[]) => xs.reduce((s, x) => s + x.rate, 0) / xs.length
  return (avg(last) - avg(first)).toFixed(1)
})

const successDots = computed(() => {
  const pts = data.value.successRate7d
  if (!pts.length) return []
  return pts.map((p, i) => ({
    x: 50 + (i / Math.max(pts.length - 1, 1)) * 520,
    y: 200 - (Math.min(p.rate, 100) / 100) * 180,
  }))
})

const successLine = computed(() =>
  successDots.value.map((d) => `${d.x},${d.y}`).join(' ')
)

const successArea = computed(() => {
  if (!successDots.value.length) return ''
  const pts = successDots.value
  return `50,200 ${pts.map((d) => `${d.x},${d.y}`).join(' ')} ${pts[pts.length - 1].x},200`
})

// ============ 工具函数 ============

function fmtNum(n: number | undefined): string {
  if (n === undefined || n === null) return '0'
  if (n >= 10000) return (n / 1000).toFixed(1) + 'k'
  return n.toLocaleString()
}
function fmtMs(ms: number | undefined): string {
  const v = ms ?? 0
  if (v === 0) return '0ms'
  if (v < 1) return '<1ms'
  if (v >= 1000) return (v / 1000).toFixed(2) + 's'
  return `${Math.round(v)}ms`
}
function fmtPct(n: number | undefined): string {
  return `${(n ?? 0).toFixed(1)}%`
}
function hourLabel(h: string): string {
  return h?.slice(0, 16) || ''
}
function hourShort(h: string): string {
  return h?.slice(11, 16) || ''
}
function dateShort(d: string): string {
  return d?.slice(5, 10) || ''
}
function barPct(v: number, max: number): string {
  if (max <= 0) return '0%'
  return `${Math.min(100, (v / max) * 100)}%`
}
function pctBar(v: number, max: number): number {
  if (max <= 0) return 0
  return Math.min(100, (v / max) * 100)
}
function tabLabel(t: string): string {
  return { all: '全部', chat: '对话', search: '搜索', upload: '上传' }[t] || t
}
const errIcon = computed(() => {
  const r = data.value.errorRate
  if (r === 0) return '✅'
  if (r < 2) return '⚠️'
  return '🔴'
})
const errColor = computed(() => {
  const r = data.value.errorRate
  if (r === 0) return 'dash-card--good'
  if (r < 2) return 'dash-card--warn'
  return 'dash-card--bad'
})

function sparkPoints(kind: 'chat'|'search'|'upload'): string {
  // 从 24h 中每 3h 取一个点，画 sparkline
  const arr = data.value.hourly
  if (!arr.length) return ''
  const step = Math.max(1, Math.floor(arr.length / 12))
  const pts: number[] = []
  for (let i = 0; i < arr.length; i += step) {
    const v = arr[i]
    const val = kind === 'chat' ? v.chatCnt : kind === 'search' ? v.searchCnt : v.uploadCnt
    pts.push(val)
  }
  const max = Math.max(...pts, 1)
  return pts.map((v, i) => `${(i / Math.max(pts.length - 1, 1)) * 80},${28 - (v / max) * 26}`).join(' ')
}

function tickClock() {
  const d = new Date()
  updatedAt.value = `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

// ============ 数据获取 ============

async function fetchData() {
  loading.value = true
  try {
    const res = await fetch(`${REPORTER_BASE}/api/v1/analytics/dashboard`, { cache: 'no-store' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const json = await res.json() as DashboardData
    if (isEmptyData(json)) {
      // 后端有响应但无数据 → 切到 mock
      data.value = genMockData()
      usingMock.value = true
    } else {
      data.value = json
      usingMock.value = false
    }
  } catch {
    // 网络失败/Reporter 未启动 → 用 mock
    data.value = genMockData()
    usingMock.value = true
  } finally {
    loading.value = false
    tickClock()
  }
}

function isEmptyData(d: DashboardData): boolean {
  return d.chatTotal === 0 && d.searchTotal === 0 && d.uploadTotal === 0 &&
         (!d.hourly || d.hourly.length === 0)
}

// ============ 生命周期 ============

onMounted(() => {
  fetchData()
  tickClock()
  timer = window.setInterval(() => {
    tickClock()
    fetchData()
  }, 30_000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
/* ================================================================
   大盘全局
   ================================================================ */
.dash {
  max-width: 1280px;
  margin: 0 auto;
  padding: 24px 24px 60px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  color: #1e293b;
}

/* ---------- 顶部 ---------- */
.dash-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}
.dash-header-left {
  display: flex; align-items: center; gap: 10px;
}
.dash-title {
  margin: 0; font-size: 22px; font-weight: 800;
}
.dash-badge {
  display: inline-block;
  font-size: 11px; padding: 2px 8px;
  border-radius: 10px; font-weight: 600;
  background: #fef3c7; color: #b45309;
  border: 1px solid #fde68a;
}
.dash-badge--live {
  background: #dcfce7; color: #15803d;
  border-color: #bbf7d0;
}
.dash-header-right {
  display: flex; align-items: center; gap: 12px;
}
.dash-updated {
  font-size: 11px; color: #94a3b8; font-variant-numeric: tabular-nums;
}
.dash-refresh {
  width: 32px; height: 32px; border-radius: 50%;
  border: 1px solid #e2e8f0; background: #fff;
  font-size: 18px; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all .2s;
}
.dash-refresh:hover { border-color: #4a6cf7; color: #4a6cf7; }
.dash-refresh.spinning { animation: spin .6s linear; }
@keyframes spin { to { transform: rotate(360deg); } }

/* ---------- 概览卡片 ---------- */
.dash-cards {
  display: grid; grid-template-columns: repeat(4, 1fr);
  gap: 16px; margin-bottom: 24px;
}
.dash-card {
  background: #fff; border-radius: 12px; padding: 18px 18px 14px;
  box-shadow: 0 1px 3px rgba(0,0,0,.06);
  display: grid; grid-template-columns: auto 1fr auto;
  gap: 12px; align-items: flex-start;
  border-left: 4px solid #e2e8f0;
  transition: transform .15s, box-shadow .15s;
  cursor: default;
  position: relative;
}
.dash-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,.08);
}
.dash-card--chat { border-left-color: #4a6cf7; }
.dash-card--search { border-left-color: #f59e0b; }
.dash-card--upload { border-left-color: #22c55e; }
.dash-card--good { border-left-color: #22c55e; }
.dash-card--warn { border-left-color: #f59e0b; }
.dash-card--bad { border-left-color: #ef4444; }
.dash-card-icon {
  font-size: 26px; line-height: 1;
  width: 40px; height: 40px;
  display: flex; align-items: center; justify-content: center;
  background: #f8fafc; border-radius: 10px;
  flex-shrink: 0;
}
.dash-card--chat .dash-card-icon { background: #eef2ff; }
.dash-card--search .dash-card-icon { background: #fef3c7; }
.dash-card--upload .dash-card-icon { background: #dcfce7; }
.dash-card-body { min-width: 0; }
.dash-card-val {
  font-size: 24px; font-weight: 800; line-height: 1.2;
  font-variant-numeric: tabular-nums;
}
.dash-card-label { font-size: 12px; color: #64748b; margin-top: 2px; }
.dash-card-sub {
  font-size: 11px; color: #94a3b8; margin-top: 4px;
  white-space: nowrap; font-variant-numeric: tabular-nums;
}
.dash-trend.up { color: #22c55e; font-weight: 600; }
.dash-trend.down { color: #ef4444; font-weight: 600; }
.dash-card-spark {
  width: 80px; height: 30px;
  align-self: center;
  opacity: 0.85;
}

/* ---------- 图表区域 ---------- */
.dash-charts {
  display: grid; grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.dash-panel {
  background: #fff; border-radius: 12px; padding: 20px 24px;
  box-shadow: 0 1px 3px rgba(0,0,0,.06);
}
.dash-panel--wide { grid-column: span 2; }
.dash-panel-head {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 12px;
}
.dash-panel-title {
  margin: 0; font-size: 14px; font-weight: 700; color: #1e293b;
}
.dash-panel-meta { font-size: 11px; color: #94a3b8; }
.dash-panel-tabs {
  display: flex; gap: 4px;
  background: #f1f5f9; padding: 2px; border-radius: 6px;
}
.dash-panel-tabs button {
  border: 0; background: transparent; padding: 4px 10px;
  font-size: 11px; color: #64748b; border-radius: 4px;
  cursor: pointer; transition: all .15s;
}
.dash-panel-tabs button.active {
  background: #fff; color: #4a6cf7; font-weight: 600;
  box-shadow: 0 1px 2px rgba(0,0,0,.08);
}
.dash-stat-pill {
  font-size: 11px; padding: 2px 8px; border-radius: 10px;
  background: #f1f5f9; color: #64748b; font-weight: 600;
}
.dash-stat-pill--up { background: #dcfce7; color: #15803d; }

/* ---------- 柱状图 ---------- */
.dash-bars {
  display: flex; align-items: flex-end;
  gap: 2px; height: 180px; padding: 0 8px;
  position: relative;
}
.dash-bar-group {
  flex: 1; display: flex; flex-direction: column; align-items: center;
  height: 100%; cursor: default; position: relative;
}
.dash-bar-stack {
  flex: 1; width: 100%;
  display: flex; flex-direction: column-reverse;
  gap: 1px;
}
.dash-bar {
  width: 100%; border-radius: 2px;
  transition: opacity .15s, filter .15s;
  animation: barGrow .8s cubic-bezier(.22,1,.36,1) backwards;
  min-height: 1px;
}
@keyframes barGrow {
  from { transform: scaleY(0); opacity: 0; }
  to { transform: scaleY(1); opacity: 1; }
}
.dash-bar:hover { filter: brightness(1.15); }
.dash-bar--chat { background: linear-gradient(180deg, #4a6cf7, #3b5bdb); }
.dash-bar--search { background: linear-gradient(180deg, #f59e0b, #d97706); }
.dash-bar--upload { background: linear-gradient(180deg, #22c55e, #16a34a); }
.dash-bar-time {
  font-size: 9px; color: #94a3b8;
  margin-top: 6px; white-space: nowrap;
}
.dash-bar-tip {
  position: absolute; top: -56px; left: 50%;
  transform: translateX(-50%);
  background: #1e293b; color: #fff;
  font-size: 11px; padding: 6px 10px;
  border-radius: 6px; white-space: nowrap;
  line-height: 1.5;
  z-index: 10;
  animation: tipIn .15s ease-out;
}
.dash-bar-tip::after {
  content: ''; position: absolute;
  bottom: -4px; left: 50%;
  transform: translateX(-50%) rotate(45deg);
  width: 8px; height: 8px;
  background: #1e293b;
}
@keyframes tipIn { from { opacity: 0; transform: translate(-50%, -4px); } }

/* ---------- 环形图 ---------- */
.dash-donut-wrap {
  display: flex; flex-direction: column; align-items: center; gap: 12px;
}
.dash-donut-container {
  position: relative; width: 160px; height: 160px;
  flex-shrink: 0;
}
.dash-donut { width: 100%; height: 100%; }
.dash-donut-center {
  position: absolute; inset: 0;
  display: flex; flex-direction: column;
  align-items: center; justify-content: center;
}
.dash-donut-total {
  font-size: 22px; font-weight: 800;
  font-variant-numeric: tabular-nums;
}
.dash-donut-label { font-size: 10px; color: #94a3b8; }

/* ---------- 图例 ---------- */
.dash-legend {
  display: flex; gap: 14px; justify-content: center;
  font-size: 12px; color: #64748b; margin-top: 8px;
}
.dash-legend--col {
  flex-direction: column; align-items: stretch;
  gap: 4px; width: 100%;
}
.dash-legend-row {
  display: grid;
  grid-template-columns: 12px 1fr auto;
  gap: 8px; align-items: center;
  padding: 4px 6px; border-radius: 4px;
  transition: background .15s;
  cursor: default;
}
.dash-legend-row:hover { background: #f8fafc; }
.dash-legend-dot {
  display: inline-block; width: 10px; height: 10px;
  border-radius: 2px;
}
.dash-legend-dot--chat { background: #4a6cf7; }
.dash-legend-dot--search { background: #f59e0b; }
.dash-legend-dot--upload { background: #22c55e; }
.dash-legend-name { color: #475569; font-weight: 500; }
.dash-legend-val { color: #94a3b8; font-variant-numeric: tabular-nums; }

/* ---------- 性能分位 ---------- */
.dash-percentile { display: flex; flex-direction: column; gap: 10px; }
.dash-percentile-row {
  display: grid; grid-template-columns: 40px 1fr 60px;
  gap: 10px; align-items: center;
  padding: 4px 6px; border-radius: 4px;
  transition: background .15s;
}
.dash-percentile-row.active { background: #f8fafc; }
.dash-percentile-label {
  font-size: 12px; font-weight: 700; color: #475569;
  font-variant-numeric: tabular-nums;
}
.dash-percentile-bar {
  height: 8px; background: #f1f5f9; border-radius: 4px; overflow: hidden;
}
.dash-percentile-fill {
  height: 100%; border-radius: 4px;
  animation: pctFill .8s cubic-bezier(.22,1,.36,1) backwards;
  transition: width .3s;
}
@keyframes pctFill { from { width: 0; } }
.dash-percentile-val {
  font-size: 12px; color: #475569; text-align: right;
  font-variant-numeric: tabular-nums; font-weight: 600;
}
.dash-percentile-hint {
  font-size: 11px; color: #94a3b8; margin-top: 8px;
  padding: 8px 10px; background: #f8fafc; border-radius: 6px;
}

/* ---------- SVG 折线图 ---------- */
.dash-line-chart { width: 100%; height: auto; display: block; }
.dash-line-pt { cursor: pointer; }
.dash-line-pt circle:nth-child(2) {
  transition: r .15s;
}
.dash-line-pt:hover circle:nth-child(2) { r: 20; }

/* ---------- 错误卡片 ---------- */
.dash-err-grid {
  display: grid; grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}
.dash-err-card {
  display: flex; align-items: center; gap: 10px;
  padding: 12px; background: #fef2f2;
  border-radius: 8px; border-left: 3px solid #ef4444;
  transition: transform .15s;
}
.dash-err-card:hover { transform: translateY(-1px); }
.dash-err-icon { font-size: 22px; flex-shrink: 0; }
.dash-err-body { flex: 1; min-width: 0; }
.dash-err-type {
  font-size: 12px; font-weight: 700; color: #991b1b;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.dash-err-msg {
  font-size: 11px; color: #64748b; margin-top: 2px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.dash-err-count {
  font-size: 16px; font-weight: 800; color: #ef4444;
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}

/* ---------- 响应式 ---------- */
@media (max-width: 1100px) {
  .dash-err-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 900px) {
  .dash-cards { grid-template-columns: repeat(2, 1fr); }
  .dash-charts { grid-template-columns: 1fr; }
  .dash-panel--wide { grid-column: span 1; }
  .dash-bars { height: 140px; }
  .dash-card-spark { display: none; }
}
@media (max-width: 500px) {
  .dash-cards { grid-template-columns: 1fr; }
  .dash { padding: 16px 12px 60px; }
  .dash-err-grid { grid-template-columns: 1fr; }
}
</style>
