<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { usePoolsStore } from '../stores/pools'
import { useGroupsStore } from '../stores/groups'
import { useAuthStore } from '../stores/auth'
import { formatPoints, timeAgo } from '../utils/format'

const route = useRoute()
const poolsStore = usePoolsStore()
const groupsStore = useGroupsStore()
const authStore = useAuthStore()

const groupId = route.params.id as string
const poolId = route.params.pid as string

const selectedOption = ref('')
const betAmount = ref(100)
const resolveOptionId = ref('')
const error = ref('')
const submitting = ref(false)

onMounted(async () => {
  await Promise.all([
    poolsStore.fetchPool(groupId, poolId),
    groupsStore.fetchGroup(groupId),
  ])
})

const pool = computed(() => poolsStore.activePool)

const canManage = computed(() => {
  if (!pool.value || !authStore.user) return false
  const isCreator = pool.value.created_by === authStore.user.id
  const member = groupsStore.activeGroup?.members?.find(m => m.user_id === authStore.user!.id)
  const isAdmin = member?.role === 'admin'
  return isCreator || isAdmin
})

const myBet = computed(() => {
  if (!pool.value || !authStore.user) return null
  return pool.value.bets?.find(b => b.user_id === authStore.user!.id)
})

const myBalance = computed(() => {
  if (!groupsStore.activeGroup || !authStore.user) return 0
  const member = groupsStore.activeGroup.members?.find(m => m.user_id === authStore.user!.id)
  return member?.points_balance || 0
})

const winningOption = computed(() => {
  if (!pool.value?.winning_option_id) return null
  return pool.value.options?.find(o => o.id === pool.value!.winning_option_id)
})

// Per-option stats
function optionBets(optionId: string) {
  return pool.value?.bets?.filter(b => b.option_id === optionId) || []
}

function optionTotal(optionId: string) {
  return optionBets(optionId).reduce((sum, b) => sum + b.points_wagered, 0)
}

async function placeBet() {
  if (!selectedOption.value) {
    error.value = 'Pick an option'
    return
  }
  if (betAmount.value < 1) {
    error.value = 'Bet at least 1 point'
    return
  }
  submitting.value = true
  error.value = ''
  try {
    await poolsStore.placeBet(groupId, poolId, selectedOption.value, betAmount.value)
    await groupsStore.fetchGroup(groupId)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Failed to place bet'
  } finally {
    submitting.value = false
  }
}

async function lockPool() {
  try {
    await poolsStore.lockPool(groupId, poolId)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Failed to lock pool'
  }
}

async function resolvePool() {
  if (!resolveOptionId.value) {
    error.value = 'Select the winning option'
    return
  }
  try {
    await poolsStore.resolvePool(groupId, poolId, resolveOptionId.value)
    await groupsStore.fetchGroup(groupId)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Failed to resolve pool'
  }
}

async function cancelPool() {
  try {
    await poolsStore.cancelPool(groupId, poolId)
    await groupsStore.fetchGroup(groupId)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Failed to cancel pool'
  }
}
</script>

<template>
  <div
    v-if="pool"
    class="max-w-2xl mx-auto"
  >
    <!-- Header -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6 mb-6">
      <div class="flex items-start justify-between mb-2">
        <h1 class="text-2xl font-bold">
          {{ pool.title }}
        </h1>
        <span
          class="text-xs font-medium px-2 py-1 rounded-full"
          :class="{
            'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400': pool.status === 'open',
            'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400': pool.status === 'locked',
            'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400': pool.status === 'resolved',
            'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400': pool.status === 'cancelled',
          }"
        >
          {{ pool.status.charAt(0).toUpperCase() + pool.status.slice(1) }}
        </span>
      </div>
      <p
        v-if="pool.description"
        class="text-gray-500 dark:text-gray-400 mb-3"
      >
        {{ pool.description }}
      </p>
      <div class="flex gap-4 text-sm text-gray-400 dark:text-gray-500">
        <span>Created by {{ pool.creator?.name }}</span>
        <span>{{ timeAgo(pool.created_at) }}</span>
        <span>{{ formatPoints(pool.total_pot) }} pts pot</span>
      </div>
    </div>

    <!-- Winning banner -->
    <div
      v-if="pool.status === 'resolved' && winningOption"
      class="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-xl p-4 mb-6 text-center"
    >
      <p class="text-green-700 dark:text-green-400 font-semibold">
        Winner: {{ winningOption.label }}
      </p>
    </div>

    <!-- Options with bet stats -->
    <div class="space-y-3 mb-6">
      <div
        v-for="option in pool.options"
        :key="option.id"
        class="bg-white dark:bg-gray-800 rounded-xl border p-4"
        :class="{
          'border-green-400 dark:border-green-600': pool.status === 'resolved' && option.id === pool.winning_option_id,
          'border-blue-400 dark:border-blue-600': myBet?.option_id === option.id && pool.status !== 'resolved',
          'border-gray-200 dark:border-gray-700': myBet?.option_id !== option.id && option.id !== pool.winning_option_id,
        }"
      >
        <div class="flex items-center justify-between">
          <div>
            <span class="font-medium">{{ option.label }}</span>
            <span
              v-if="myBet?.option_id === option.id"
              class="ml-2 text-xs text-blue-500"
            >Your pick</span>
          </div>
          <div class="text-sm text-gray-500 dark:text-gray-400">
            {{ optionBets(option.id).length }} bets, {{ formatPoints(optionTotal(option.id)) }} pts
          </div>
        </div>
        <!-- Show bettors on resolved pools -->
        <div
          v-if="pool.status === 'resolved' || pool.status === 'locked'"
          class="mt-2 flex flex-wrap gap-1"
        >
          <span
            v-for="bet in optionBets(option.id)"
            :key="bet.id"
            class="text-xs bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded-full text-gray-600 dark:text-gray-300"
          >
            {{ bet.user?.name }} ({{ formatPoints(bet.points_wagered) }})
          </span>
        </div>
      </div>
    </div>

    <!-- Place bet form (only if open and haven't bet) -->
    <div
      v-if="pool.status === 'open' && !myBet"
      class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6 mb-6"
    >
      <h2 class="text-lg font-semibold mb-4">
        Place Your Bet
      </h2>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">
        Your balance: {{ formatPoints(myBalance) }} pts
      </p>

      <div class="space-y-3">
        <div>
          <label class="block text-sm font-medium mb-1">Pick an option</label>
          <select
            v-model="selectedOption"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-800 dark:text-white"
          >
            <option value="">
              Select...
            </option>
            <option
              v-for="option in pool.options"
              :key="option.id"
              :value="option.id"
            >
              {{ option.label }}
            </option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium mb-1">Points to wager</label>
          <input
            v-model.number="betAmount"
            type="number"
            min="1"
            :max="myBalance"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-800 dark:text-white"
          >
        </div>

        <p
          v-if="error"
          class="text-red-500 dark:text-red-400 text-sm"
        >
          {{ error }}
        </p>

        <button
          :disabled="submitting"
          class="w-full py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 font-medium"
          @click="placeBet"
        >
          {{ submitting ? 'Placing...' : 'Place Bet' }}
        </button>
      </div>
    </div>

    <!-- Your bet info -->
    <div
      v-if="myBet"
      class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-xl p-4 mb-6"
    >
      <p class="text-blue-700 dark:text-blue-400 font-medium">
        You bet {{ formatPoints(myBet.points_wagered) }} pts on "{{ myBet.option?.label }}"
      </p>
    </div>

    <!-- Admin/Creator controls -->
    <div
      v-if="canManage && (pool.status === 'open' || pool.status === 'locked')"
      class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6"
    >
      <h2 class="text-lg font-semibold mb-4">
        Manage Pool
      </h2>
      <div class="space-y-3">
        <button
          v-if="pool.status === 'open'"
          class="w-full py-2 bg-yellow-500 text-white rounded-lg hover:bg-yellow-600 font-medium"
          @click="lockPool"
        >
          Lock Pool (no more bets)
        </button>

        <div v-if="pool.status === 'open' || pool.status === 'locked'">
          <label class="block text-sm font-medium mb-1">Resolve: Pick the winner</label>
          <div class="flex gap-2">
            <select
              v-model="resolveOptionId"
              class="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
            >
              <option value="">
                Select winning option...
              </option>
              <option
                v-for="option in pool.options"
                :key="option.id"
                :value="option.id"
              >
                {{ option.label }}
              </option>
            </select>
            <button
              class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 font-medium"
              @click="resolvePool"
            >
              Resolve
            </button>
          </div>
        </div>

        <button
          class="w-full py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 font-medium"
          @click="cancelPool"
        >
          Cancel Pool (refund all)
        </button>
      </div>
    </div>
  </div>
</template>
