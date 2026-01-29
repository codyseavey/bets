<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '../services/api'
import { formatPoints } from '../utils/format'

const route = useRoute()
const groupId = route.params.id as string

interface LeaderboardEntry {
  user_id: string
  name: string
  avatar_url: string
  points_balance: number
  total_wins: number
  total_losses: number
  total_bets: number
  rank: number
}

const entries = ref<LeaderboardEntry[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    const { data } = await api.get(`/groups/${groupId}/leaderboard`)
    entries.value = data
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="max-w-2xl mx-auto">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold">
        Leaderboard
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
      class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden"
    >
      <table class="w-full">
        <thead>
          <tr class="border-b border-gray-200 dark:border-gray-700 text-left text-xs text-gray-500 dark:text-gray-400 uppercase">
            <th class="px-4 py-3">
              Rank
            </th>
            <th class="px-4 py-3">
              Player
            </th>
            <th class="px-4 py-3 text-right">
              Points
            </th>
            <th class="px-4 py-3 text-right hidden sm:table-cell">
              W/L
            </th>
            <th class="px-4 py-3 text-right hidden sm:table-cell">
              Bets
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="entry in entries"
            :key="entry.user_id"
            class="border-b border-gray-100 dark:border-gray-700/50 last:border-0"
          >
            <td class="px-4 py-3 font-bold text-lg">
              <span
                :class="{
                  'text-yellow-500': entry.rank === 1,
                  'text-gray-400': entry.rank === 2,
                  'text-amber-600': entry.rank === 3,
                }"
              >
                {{ entry.rank }}
              </span>
            </td>
            <td class="px-4 py-3">
              <div class="flex items-center gap-2">
                <img
                  v-if="entry.avatar_url"
                  :src="entry.avatar_url"
                  :alt="entry.name"
                  class="w-8 h-8 rounded-full"
                >
                <span class="font-medium">{{ entry.name }}</span>
              </div>
            </td>
            <td class="px-4 py-3 text-right font-semibold">
              {{ formatPoints(entry.points_balance) }}
            </td>
            <td class="px-4 py-3 text-right hidden sm:table-cell text-sm">
              <span class="text-green-500">{{ entry.total_wins }}W</span>
              /
              <span class="text-red-500">{{ entry.total_losses }}L</span>
            </td>
            <td class="px-4 py-3 text-right hidden sm:table-cell text-sm text-gray-500">
              {{ entry.total_bets }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
