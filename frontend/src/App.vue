<template>
  <el-config-provider>
    <el-container class="app-layout">
      <el-header class="app-header">
        <div class="logo">
          <el-icon :size="22"><Files /></el-icon>
          <span>S3C – S3 Cloud Browser</span>
        </div>
      </el-header>
      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </el-config-provider>
</template>

<style>
html,
body,
#app {
  margin: 0;
  padding: 0;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.app-layout {
  height: 100vh;
}

.app-header {
  display: flex;
  align-items: center;
  background: #409eff;
  color: #fff;
  padding: 0 20px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 600;
}

.app-main {
  padding: 0;
  overflow: hidden;
}
</style>
