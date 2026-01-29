import { defineStore } from 'pinia'
import api from '../services/api'

export interface PoolOption {
  id: string
  pool_id: string
  label: string
  description: string
}

export interface Bet {
  id: string
  pool_id: string
  user_id: string
  option_id: string
  points_wagered: number
  created_at: string
  user: { id: string; name: string; avatar_url: string }
  option: PoolOption
}

export interface Pool {
  id: string
  group_id: string
  title: string
  description: string
  status: 'open' | 'locked' | 'resolved' | 'cancelled'
  created_by: string
  resolved_at: string | null
  created_at: string
  creator: { id: string; name: string; avatar_url: string }
  options: PoolOption[]
  bets: Bet[]
  winning_option_id: string
  total_pot: number
  bet_count: number
}

export const usePoolsStore = defineStore('pools', {
  state: () => ({
    pools: [] as Pool[],
    activePool: null as Pool | null,
    loading: false,
  }),

  actions: {
    async fetchPools(groupId: string, status?: string) {
      this.loading = true
      try {
        const params = status ? { status } : {}
        const { data } = await api.get(`/groups/${groupId}/pools`, { params })
        this.pools = data
      } finally {
        this.loading = false
      }
    },

    async fetchPool(groupId: string, poolId: string) {
      const { data } = await api.get(`/groups/${groupId}/pools/${poolId}`)
      this.activePool = data
      return data
    },

    async createPool(groupId: string, title: string, description: string, options: string[]) {
      const { data } = await api.post(`/groups/${groupId}/pools`, { title, description, options })
      this.pools.unshift(data)
      return data
    },

    async placeBet(groupId: string, poolId: string, optionId: string, points: number) {
      const { data } = await api.post(`/groups/${groupId}/pools/${poolId}/bet`, {
        option_id: optionId,
        points,
      })
      await this.fetchPool(groupId, poolId)
      return data
    },

    async lockPool(groupId: string, poolId: string) {
      await api.post(`/groups/${groupId}/pools/${poolId}/lock`)
      await this.fetchPool(groupId, poolId)
    },

    async resolvePool(groupId: string, poolId: string, winningOptionId: string) {
      await api.post(`/groups/${groupId}/pools/${poolId}/resolve`, {
        winning_option_id: winningOptionId,
      })
      await this.fetchPool(groupId, poolId)
    },

    async cancelPool(groupId: string, poolId: string) {
      await api.post(`/groups/${groupId}/pools/${poolId}/cancel`)
      await this.fetchPool(groupId, poolId)
    },
  },
})
