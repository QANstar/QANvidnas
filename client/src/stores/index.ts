import { create } from 'zustand';

interface User {
  id: number;
  username: string;
  isAdmin: boolean;
}

interface PlaylistMedia {
  id: number;
  title: string;
  type: string;
  coverPath?: string;
}

interface PlayerState {
  currentMedia: PlaylistMedia | null;
  playlistId: number | null;
  playlistItems: PlaylistMedia[];
  currentIndex: number;
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
  setPlaylistItems: (items: PlaylistMedia[], startIndex?: number) => void;
  playNext: () => PlaylistMedia | null;
  playPrevious: () => PlaylistMedia | null;
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
    playlistItems: [],
    currentIndex: -1,
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
  setPlaylistItems: (items, startIndex = 0) => set((s) => ({
    player: { ...s.player, playlistItems: items, currentIndex: startIndex },
  })),
  playNext: () => {
    let nextMedia: PlaylistMedia | null = null;
    set((s) => {
      const { playlistItems, currentIndex, playMode } = s.player;
      if (playlistItems.length === 0) return s;
      let nextIndex: number;
      if (playMode === 'random') {
        nextIndex = Math.floor(Math.random() * playlistItems.length);
      } else if (playMode === 'single-loop') {
        nextIndex = currentIndex;
      } else if (playMode === 'loop') {
        nextIndex = (currentIndex + 1) % playlistItems.length;
      } else {
        // sequential
        nextIndex = currentIndex + 1;
        if (nextIndex >= playlistItems.length) return s; // end of playlist
      }
      nextMedia = playlistItems[nextIndex];
      return {
        player: {
          ...s.player,
          currentMedia: nextMedia,
          currentIndex: nextIndex,
          isPlaying: true,
          currentTime: 0,
          duration: 0,
        },
      };
    });
    return nextMedia;
  },
  playPrevious: () => {
    let prevMedia: PlaylistMedia | null = null;
    set((s) => {
      const { playlistItems, currentIndex, playMode } = s.player;
      if (playlistItems.length === 0) return s;
      let prevIndex: number;
      if (playMode === 'random') {
        prevIndex = Math.floor(Math.random() * playlistItems.length);
      } else if (playMode === 'single-loop') {
        prevIndex = currentIndex;
      } else {
        prevIndex = currentIndex - 1;
        if (prevIndex < 0) {
          if (playMode === 'loop') {
            prevIndex = playlistItems.length - 1;
          } else {
            return s; // beginning of playlist
          }
        }
      }
      prevMedia = playlistItems[prevIndex];
      return {
        player: {
          ...s.player,
          currentMedia: prevMedia,
          currentIndex: prevIndex,
          isPlaying: true,
          currentTime: 0,
          duration: 0,
        },
      };
    });
    return prevMedia;
  },
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
