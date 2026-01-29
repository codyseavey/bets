<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '../services/api'
import { timeAgo, formatPoints } from '../utils/format'

const route = useRoute()
const groupId = route.params.id as string

interface PointsLogEntry {
  id: string
  group_id: string
  user_id: string
  amount: number
  type: string
  reference_id: string
  note: string
  created_at: string
  user: { id: string; name: string; avatar_url: string }
}

const items = ref<PointsLogEntry[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)

async function fetchHistory() {
  loading.value = true
  try {
    const { data } = await api.get(`/groups/${groupId}/history`, {
      params: { page: page.value, limit: 50 },
    })
    items.value = data.items || []
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function loadMore() {
  page.value++
  fetchMore()
}

async function fetchMore() {
  const { data } = await api.get(`/groups/${groupId}/history`, {
    params: { page: page.value, limit: 50 },
  })
  items.value.push(...(data.items || []))
}

const typeConfig: Record<string, { label: string; color: string }> = {
  initial: { label: 'Joined', color: 'text-blue-500' },
  admin_grant: { label: 'Granted', color: 'text-green-500' },
  bet_placed: { label: 'Bet', color: 'text-red-500' },
  bet_won: { label: 'Won', color: 'text-green-500' },
  bet_refund: { label: 'Refund', color: 'text-yellow-500' },
  pool_resolved: { label: 'Resolved', color: 'text-blue-500' },
}

onMounted(fetchHistory)
</script>

<template>
  <div class="max-w-2xl mx-auto">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold">
        Points History
      </h1>
      <router-link
        :to="`/groups/${groupId}`"
        class="text-sm text-gray-500 dark:text-gray-400 hover:underline"
      >
        Back to Group
      </router-link>
    </div>

    <div
      v-if="loading"
      class="text-center py-8 text-gray-500"
    >
      Loading...
    </div>

    <div
      v-else
      class="space-y-2"
    >
      <div
        v-for="item in items"
        :key="item.id"
        class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 flex items-center gap-3"
      >
        <img
          v-if="item.user?.avatar_url"
          :src="item.user.avatar_url"
          :alt="item.user.name"
          class="w-8 h-8 rounded-full shrink-0"
        >
        <div class="flex-1 min-w-0">
          <p class="text-sm">
            <span class="font-medium">{{ item.user?.name }}</span>
            <span
              class="ml-1 text-xs font-medium"
              :class="typeConfig[item.type]?.color || 'text-gray-500'"
            >
              {{ typeConfig[item.type]?.label || item.type }}
            </span>
          </p>
          <p
            v-if="item.note"
            class="text-xs text-gray-500 dark:text-gray-400 truncate"
          >
            {{ item.note }}
          </p>
        </div>
        <div class="text-right shrink-0">
          <p
            class="font-semibold text-sm"
            :class="item.amount >= 0 ? 'text-green-500' : 'text-red-500'"
          >
            {{ item.amount >= 0 ? '+' : '' }}{{ formatPoints(item.amount) }}
          </p>
          <p class="text-xs text-gray-400">
            {{ timeAgo(item.created_at) }}
          </p>
        </div>
      </div>

      <button
        v-if="items.length < total"
        class="w-full py-2 text-sm text-blue-600 dark:text-blue-400 hover:underline"
        @click="loadMore"
      >
        Load More
      </button>
    </div>
  </div>
</template>
