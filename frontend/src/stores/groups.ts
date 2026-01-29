import { defineStore } from 'pinia'
import api from '../services/api'

export interface GroupMember {
  group_id: string
  user_id: string
  role: string
  points_balance: number
  joined_at: string
  user: {
    id: string
    name: string
    email: string
    avatar_url: string
  }
}

export interface Group {
  id: string
  name: string
  invite_code: string
  default_points: number
  created_by: string
  created_at: string
  creator: { id: string; name: string; avatar_url: string }
  members: GroupMember[]
}

export const useGroupsStore = defineStore('groups', {
  state: () => ({
    groups: [] as Group[],
    activeGroup: null as Group | null,
    loading: false,
  }),

  actions: {
    async fetchGroups() {
      this.loading = true
      try {
        const { data } = await api.get('/groups')
        this.groups = data
      } finally {
        this.loading = false
      }
    },

    async fetchGroup(id: string) {
      const { data } = await api.get(`/groups/${id}`)
      this.activeGroup = data
      return data
    },

    async createGroup(name: string, defaultPoints: number) {
      const { data } = await api.post('/groups', { name, default_points: defaultPoints })
      this.groups.push(data)
      return data
    },

    async joinGroup(inviteCode: string) {
      const { data } = await api.post('/groups/join', { invite_code: inviteCode })
      await this.fetchGroups()
      return data
    },

    async updateGroup(id: string, name: string, defaultPoints: number) {
      await api.put(`/groups/${id}`, { name, default_points: defaultPoints })
      await this.fetchGroup(id)
    },

    async grantPoints(groupId: string, userId: string, amount: number, note: string) {
      await api.post(`/groups/${groupId}/grant`, { user_id: userId, amount, note })
      await this.fetchGroup(groupId)
    },

    async kickMember(groupId: string, userId: string) {
      await api.delete(`/groups/${groupId}/members/${userId}`)
      await this.fetchGroup(groupId)
    },

    async regenerateInvite(groupId: string) {
      const { data } = await api.post(`/groups/${groupId}/regenerate-invite`)
      if (this.activeGroup && this.activeGroup.id === groupId) {
        this.activeGroup.invite_code = data.invite_code
      }
      return data.invite_code
    },
  },
})
