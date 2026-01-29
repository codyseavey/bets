<script setup lang="ts">
import { onMounted } from 'vue'
import { useGroupsStore } from '../stores/groups'
import GroupCard from '../components/GroupCard.vue'

const groupsStore = useGroupsStore()

onMounted(() => {
  groupsStore.fetchGroups()
})
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold">
        Your Groups
      </h1>
      <div class="flex gap-2">
        <router-link
          to="/groups/join"
          class="px-4 py-2 text-sm bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600"
        >
          Join Group
        </router-link>
        <router-link
          to="/groups/new"
          class="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          New Group
        </router-link>
      </div>
    </div>

    <div
      v-if="groupsStore.loading"
      class="text-center py-12 text-gray-500 dark:text-gray-400"
    >
      Loading groups...
    </div>

    <div
      v-else-if="groupsStore.groups.length === 0"
      class="text-center py-12"
    >
      <p class="text-gray-500 dark:text-gray-400 mb-4">
        You're not in any groups yet.
      </p>
      <div class="flex gap-2 justify-center">
        <router-link
          to="/groups/new"
          class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          Create a Group
        </router-link>
        <router-link
          to="/groups/join"
          class="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 rounded-lg"
        >
          Join with Code
        </router-link>
      </div>
    </div>

    <div
      v-else
      class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4"
    >
      <GroupCard
        v-for="group in groupsStore.groups"
        :key="group.id"
        :group="group"
      />
    </div>
  </div>
</template>
