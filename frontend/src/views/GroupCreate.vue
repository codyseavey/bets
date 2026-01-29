<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useGroupsStore } from '../stores/groups'

const router = useRouter()
const groupsStore = useGroupsStore()

const name = ref('')
const defaultPoints = ref(1000)
const error = ref('')
const submitting = ref(false)

async function submit() {
  if (!name.value.trim()) {
    error.value = 'Group name is required'
    return
  }
  submitting.value = true
  error.value = ''
  try {
    const group = await groupsStore.createGroup(name.value.trim(), defaultPoints.value)
    router.push(`/groups/${group.id}`)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Failed to create group'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="max-w-lg mx-auto">
    <h1 class="text-2xl font-bold mb-6">
      Create a Group
    </h1>

    <form
      class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6 space-y-4"
      @submit.prevent="submit"
    >
      <div>
        <label class="block text-sm font-medium mb-1">Group Name</label>
        <input
          v-model="name"
          type="text"
          placeholder="e.g. Office Pool, Family Bets"
          class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-800 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        >
      </div>

      <div>
        <label class="block text-sm font-medium mb-1">Default Starting Points</label>
        <input
          v-model.number="defaultPoints"
          type="number"
          min="1"
          class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-800 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        >
        <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
          Everyone who joins gets this many points to start.
        </p>
      </div>

      <p
        v-if="error"
        class="text-red-500 dark:text-red-400 text-sm"
      >
        {{ error }}
      </p>

      <button
        type="submit"
        :disabled="submitting"
        class="w-full py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 font-medium"
      >
        {{ submitting ? 'Creating...' : 'Create Group' }}
      </button>
    </form>
  </div>
</template>
