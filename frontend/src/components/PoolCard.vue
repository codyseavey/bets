<script setup lang="ts">
import type { Pool } from '../stores/pools'
import { timeAgo, formatPoints } from '../utils/format'
import { computed } from 'vue'

const props = defineProps<{
  pool: Pool
  groupId: string
}>()

const statusConfig = computed(() => {
  const map: Record<string, { label: string; classes: string }> = {
    open: { label: 'Open', classes: 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400' },
    locked: { label: 'Locked', classes: 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400' },
    resolved: { label: 'Resolved', classes: 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400' },
    cancelled: { label: 'Cancelled', classes: 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400' },
  }
  return map[props.pool.status] || map.open
})
</script>

<template>
  <router-link
    :to="`/groups/${groupId}/pools/${pool.id}`"
    class="block bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-5 hover:shadow-md transition-shadow"
  >
    <div class="flex items-start justify-between mb-2">
      <h3 class="text-lg font-semibold">
        {{ pool.title }}
      </h3>
      <span
        class="text-xs font-medium px-2 py-0.5 rounded-full shrink-0 ml-2"
        :class="statusConfig.classes"
      >
        {{ statusConfig.label }}
      </span>
    </div>
    <p
      v-if="pool.description"
      class="text-sm text-gray-500 dark:text-gray-400 mb-3"
    >
      {{ pool.description }}
    </p>
    <div class="flex items-center gap-4 text-xs text-gray-400 dark:text-gray-500">
      <span>{{ pool.bet_count }} bets</span>
      <span>{{ formatPoints(pool.total_pot) }} pts pot</span>
      <span>{{ pool.options?.length || 0 }} options</span>
      <span class="ml-auto">{{ timeAgo(pool.created_at) }}</span>
    </div>
  </router-link>
</template>
