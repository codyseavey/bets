<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useGroupsStore } from '../stores/groups'

const router = useRouter()
const route = useRoute()
const groupsStore = useGroupsStore()

const inviteCode = ref('')
const error = ref('')
const submitting = ref(false)

onMounted(() => {
  // Support direct invite links like /groups/join/ABC123
  if (route.params.code) {
    inviteCode.value = route.params.code as string
    submit()
  }
})

async function submit() {
  if (!inviteCode.value.trim()) {
    error.value = 'Invite code is required'
    return
  }
  submitting.value = true
  error.value = ''
  try {
    const group = await groupsStore.joinGroup(inviteCode.value.trim().toUpperCase())
    router.push(`/groups/${group.id}`)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Failed to join group'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="max-w-lg mx-auto">
    <h1 class="text-2xl font-bold mb-6">
      Join a Group
    </h1>

    <form
      class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6 space-y-4"
      @submit.prevent="submit"
    >
      <div>
        <label class="block text-sm font-medium mb-1">Invite Code</label>
        <input
          v-model="inviteCode"
          type="text"
          placeholder="Enter invite code"
          class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-800 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent uppercase tracking-widest text-center text-lg font-mono"
          maxlength="8"
        >
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
        {{ submitting ? 'Joining...' : 'Join Group' }}
      </button>
    </form>
  </div>
</template>
