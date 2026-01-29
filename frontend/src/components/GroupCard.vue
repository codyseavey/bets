<script setup lang="ts">
import type { Group } from '../stores/groups'
import { formatDate } from '../utils/format'

defineProps<{
  group: Group
}>()
</script>

<template>
  <router-link
    :to="`/groups/${group.id}`"
    class="block bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-5 hover:shadow-md transition-shadow"
  >
    <h3 class="text-lg font-semibold mb-1">
      {{ group.name }}
    </h3>
    <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">
      Created by {{ group.creator?.name || 'Unknown' }}
    </p>
    <div class="flex items-center justify-between text-xs text-gray-400 dark:text-gray-500">
      <span>{{ formatDate(group.created_at) }}</span>
      <span class="bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 px-2 py-0.5 rounded-full">
        {{ group.default_points.toLocaleString() }} pts
      </span>
    </div>
  </router-link>
</template>
