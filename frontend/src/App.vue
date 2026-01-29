<script setup lang="ts">
import AppNav from './components/AppNav.vue'
import { useAuthStore } from './stores/auth'

const authStore = useAuthStore()
</script>

<template>
  <div class="min-h-screen bg-gray-100 dark:bg-gray-900 text-gray-800 dark:text-white">
    <AppNav v-if="authStore.isAuthenticated" />
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <router-view />
    </main>
  </div>
</template>
