import { create } from 'zustand';

interface User {
  id: number;
  username: string;
  is_admin: boolean;
}

interface PlayerState {
  currentMedia: { id: number; title: string; type: string; cover_path?: string } | null;
  playlistId: number | null;
  playMode: 'sequential' | 'loop' | 'random' | 'single-loop';
  isPlaying: boolean;
  currentTime: number;
  duration: number;
  volume: number;
  isMuted: boolean;
}

interface FilterState {
  typeFilter: string; // 'video' | 'audio' | ''
  sortBy: string;
  order: string;
}

interface AppState {
  user: User | null;
  token: string | null;
  player: PlayerState;
  filter: FilterState;
  isTVMode: boolean;

  // Auth actions
  setAuth: (user: User, token: string) => void;
  logout: () => void;

  // Player actions
  setCurrentMedia: (media: PlayerState['currentMedia']) => void;
  setPlaylist: (playlistId: number | null) => void;
  setPlayMode: (mode: PlayerState['playMode']) => void;
  setPlaying: (playing: boolean) => void;
  setCurrentTime: (time: number) => void;
  setDuration: (duration: number) => void;
  setVolume: (volume: number) => void;
  setMuted: (muted: boolean) => void;

  // Filter actions
  setTypeFilter: (type: string) => void;
  setSortBy: (sort: string) => void;
  setOrder: (order: string) => void;

  // TV mode
  setIsTVMode: (tv: boolean) => void;
}

export const useStore = create<AppState>((set) => ({
  user: null,
  token: localStorage.getItem('token'),
  player: {
    currentMedia: null,
    playlistId: null,
    playMode: 'sequential',
    isPlaying: false,
    currentTime: 0,
    duration: 0,
    volume: 1,
    isMuted: false,
  },
  filter: {
    typeFilter: '',
    sortBy: 'created_at',
    order: 'desc',
  },
  isTVMode: false,

  setAuth: (user, token) => {
    localStorage.setItem('token', token);
    set({ user, token });
  },

  logout: () => {
    localStorage.removeItem('token');
    set({ user: null, token: null });
  },

  setCurrentMedia: (media) => set((s) => ({ player: { ...s.player, currentMedia: media } })),
  setPlaylist: (id) => set((s) => ({ player: { ...s.player, playlistId: id } })),
  setPlayMode: (mode) => set((s) => ({ player: { ...s.player, playMode: mode } })),
  setPlaying: (playing) => set((s) => ({ player: { ...s.player, isPlaying: playing } })),
  setCurrentTime: (time) => set((s) => ({ player: { ...s.player, currentTime: time } })),
  setDuration: (duration) => set((s) => ({ player: { ...s.player, duration: duration } })),
  setVolume: (volume) => set((s) => ({ player: { ...s.player, volume: volume } })),
  setMuted: (muted) => set((s) => ({ player: { ...s.player, isMuted: muted } })),

  setTypeFilter: (type) => set((s) => ({ filter: { ...s.filter, typeFilter: type } })),
  setSortBy: (sort) => set((s) => ({ filter: { ...s.filter, sortBy: sort } })),
  setOrder: (order) => set((s) => ({ filter: { ...s.filter, order: order } })),

  setIsTVMode: (tv) => set({ isTVMode: tv }),
}));
