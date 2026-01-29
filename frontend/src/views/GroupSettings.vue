<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGroupsStore } from '../stores/groups'
import { useAuthStore } from '../stores/auth'
import { formatPoints } from '../utils/format'

const route = useRoute()
const router = useRouter()
const groupsStore = useGroupsStore()
const authStore = useAuthStore()

const groupId = route.params.id as string
const name = ref('')
const defaultPoints = ref(1000)
const grantUserId = ref('')
const grantAmount = ref(500)
const grantNote = ref('')
const error = ref('')
const message = ref('')
const copied = ref(false)
const showDeleteConfirm = ref(false)

onMounted(async () => {
  await groupsStore.fetchGroup(groupId)
  if (groupsStore.activeGroup) {
    name.value = groupsStore.activeGroup.name
    defaultPoints.value = groupsStore.activeGroup.default_points
  }
})

const members = computed(() => {
  return groupsStore.activeGroup?.members?.filter(m => m.user_id !== authStore.user?.id) || []
})

const inviteCode = computed(() => groupsStore.activeGroup?.invite_code || '')
const inviteLink = computed(() => `${window.location.origin}/groups/join/${inviteCode.value}`)

async function updateGroup() {
  error.value = ''
  message.value = ''
  try {
    await groupsStore.updateGroup(groupId, name.value, defaultPoints.value)
    message.value = 'Group updated'
  } catch {
    error.value = 'Failed to update group'
  }
}

async function grantPoints() {
  if (!grantUserId.value || grantAmount.value < 1) {
    error.value = 'Select a member and amount'
    return
  }
  error.value = ''
  try {
    await groupsStore.grantPoints(groupId, grantUserId.value, grantAmount.value, grantNote.value)
    message.value = `Granted ${grantAmount.value} points`
    grantNote.value = ''
  } catch {
    error.value = 'Failed to grant points'
  }
}

async function kickMember(userId: string) {
  try {
    await groupsStore.kickMember(groupId, userId)
  } catch {
    error.value = 'Failed to kick member'
  }
}

async function regenerateInvite() {
  try {
    await groupsStore.regenerateInvite(groupId)
    message.value = 'New invite code generated'
  } catch {
    error.value = 'Failed to regenerate invite code'
  }
}

async function deleteGroup() {
  try {
    await groupsStore.deleteGroup(groupId)
    router.push('/')
  } catch {
    error.value = 'Failed to delete group'
  }
}

function copyInvite() {
  navigator.clipboard.writeText(inviteLink.value)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}
</script>

<template>
  <div class="max-w-2xl mx-auto space-y-6">
    <h1 class="text-2xl font-bold">
      Group Settings
    </h1>

    <p
      v-if="message"
      class="text-green-600 dark:text-green-400 text-sm bg-green-50 dark:bg-green-900/20 p-3 rounded-lg"
    >
      {{ message }}
    </p>
    <p
      v-if="error"
      class="text-red-500 dark:text-red-400 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg"
    >
      {{ error }}
    </p>

    <!-- General settings -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <h2 class="text-lg font-semibold mb-4">
        General
      </h2>
      <form
        class="space-y-3"
        @submit.prevent="updateGroup"
      >
        <div>
          <label class="block text-sm font-medium mb-1">Group Name</label>
          <input
            v-model="name"
            type="text"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-800 dark:text-white"
          >
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Default Points for New Members</label>
          <input
            v-model.number="defaultPoints"
            type="number"
            min="1"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-800 dark:text-white"
          >
        </div>
        <button
          type="submit"
          class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          Save
        </button>
      </form>
    </div>

    <!-- Invite code -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <h2 class="text-lg font-semibold mb-4">
        Invite Code
      </h2>
      <div class="flex items-center gap-3 mb-3">
        <code class="text-2xl font-mono tracking-widest bg-gray-100 dark:bg-gray-700 px-4 py-2 rounded-lg">
          {{ inviteCode }}
        </code>
        <button
          class="px-3 py-2 text-sm bg-gray-200 dark:bg-gray-700 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600"
          @click="copyInvite"
        >
          {{ copied ? 'Copied!' : 'Copy Link' }}
        </button>
      </div>
      <button
        class="text-sm text-red-500 hover:underline"
        @click="regenerateInvite"
      >
        Regenerate Code
      </button>
    </div>

    <!-- Grant points -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <h2 class="text-lg font-semibold mb-4">
        Grant Points
      </h2>
      <div class="space-y-3">
        <div>
          <label class="block text-sm font-medium mb-1">Member</label>
          <select
            v-model="grantUserId"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
          >
            <option value="">
              Select member...
            </option>
            <option
              v-for="member in groupsStore.activeGroup?.members"
              :key="member.user_id"
              :value="member.user_id"
            >
              {{ member.user?.name }} ({{ formatPoints(member.points_balance) }} pts)
            </option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Amount</label>
          <input
            v-model.number="grantAmount"
            type="number"
            min="1"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-800 dark:text-white"
          >
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Note (optional)</label>
          <input
            v-model="grantNote"
            type="text"
            placeholder="Reason for granting points"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-800 dark:text-white"
          >
        </div>
        <button
          class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700"
          @click="grantPoints"
        >
          Grant Points
        </button>
      </div>
    </div>

    <!-- Members -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <h2 class="text-lg font-semibold mb-4">
        Members
      </h2>
      <div class="space-y-2">
        <div
          v-for="member in members"
          :key="member.user_id"
          class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
        >
          <div class="flex items-center gap-3">
            <img
              v-if="member.user?.avatar_url"
              :src="member.user.avatar_url"
              :alt="member.user.name"
              class="w-8 h-8 rounded-full"
            >
            <div>
              <p class="font-medium text-sm">
                {{ member.user?.name }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ member.role }} - {{ formatPoints(member.points_balance) }} pts
              </p>
            </div>
          </div>
          <button
            class="text-xs text-red-500 hover:underline"
            @click="kickMember(member.user_id)"
          >
            Kick
          </button>
        </div>
      </div>
    </div>

    <!-- Danger Zone -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-red-300 dark:border-red-700 p-6">
      <h2 class="text-lg font-semibold text-red-600 dark:text-red-400 mb-4">
        Danger Zone
      </h2>
      <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
        Permanently delete this group and all its pools, bets, and history. This cannot be undone.
      </p>
      <div v-if="!showDeleteConfirm">
        <button
          class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
          @click="showDeleteConfirm = true"
        >
          Delete Group
        </button>
      </div>
      <div
        v-else
        class="flex items-center gap-3"
      >
        <button
          class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
          @click="deleteGroup"
        >
          Yes, delete permanently
        </button>
        <button
          class="px-4 py-2 bg-gray-200 dark:bg-gray-700 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600"
          @click="showDeleteConfirm = false"
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
</template>
