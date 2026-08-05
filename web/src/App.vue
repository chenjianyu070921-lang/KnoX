<template>
  <div id="app-root">
    <nav class="app-nav">
      <div class="app-nav-inner">
        <span class="app-brand">KnoX</span>
        <div class="app-nav-links">
          <router-link to="/" class="app-nav-link">💬 对话</router-link>
          <router-link to="/docs" class="app-nav-link">📄 文档</router-link>
          <router-link to="/dashboard" class="app-nav-link">📊 大盘</router-link>
        </div>
      </div>
    </nav>
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.app-nav {
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
  position: sticky;
  top: 0;
  z-index: 1000;
}
.app-nav-inner {
  max-width: 1280px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 52px;
  padding: 0 24px;
}
.app-brand {
  font-size: 20px;
  font-weight: 700;
  color: #4a6cf7;
  letter-spacing: -.5px;
}
.app-nav-links {
  display: flex;
  gap: 6px;
}
.app-nav-link {
  text-decoration: none;
  font-size: 13.5px;
  font-weight: 500;
  padding: 6px 16px;
  border-radius: 6px;
  color: #64748b;
  transition: all .15s;
}
.app-nav-link:hover {
  background: #f1f5f9;
  color: #1e293b;
}
.app-nav-link.router-link-active {
  background: #eef2ff;
  color: #4a6cf7;
  font-weight: 600;
}
.app-main {
  min-height: calc(100vh - 52px);
}
</style>
