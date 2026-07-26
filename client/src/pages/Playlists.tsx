import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { playlistAPI } from '../api/client';
import { useStore } from '../stores';
import './Playlists.css';

interface PlaylistItem {
  id: number;
  name: string;
  folderPath: string;
  playMode: string;
  createdAt: string;
}

export default function Playlists() {
  const [playlists, setPlaylists] = useState<PlaylistItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newPath, setNewPath] = useState('');
  const navigate = useNavigate();
  const { setCurrentMedia, setPlaylist, setPlaylistItems } = useStore();

  const fetchPlaylists = async () => {
    try {
      const res = await playlistAPI.list();
      setPlaylists(res.data.items);
    } catch (err) {
      console.error('Failed to load playlists:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchPlaylists(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await playlistAPI.create(newName, newPath);
      setShowCreate(false);
      setNewName('');
      setNewPath('');
      fetchPlaylists();
    } catch (err) {
      console.error('Failed to create playlist:', err);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定删除这个播放列表？')) return;
    await playlistAPI.delete(id);
    fetchPlaylists();
  };

  const handlePlay = async (playlist: PlaylistItem) => {
    try {
      const res = await playlistAPI.get(playlist.id);
      const items = res.data.items;
      if (items.length > 0) {
        const mediaItems = items.map((item: { media: { id: number; title: string; type: string; coverPath?: string } }) => item.media);
        setPlaylist(playlist.id);
        setPlaylistItems(mediaItems, 0);
        navigate('/player');
      }
    } catch (err) {
      console.error('Failed to play:', err);
    }
  };

  const playModeLabels: Record<string, string> = {
    'sequential': '顺序播放',
    'loop': '循环播放',
    'random': '随机播放',
    'single-loop': '单曲循环',
  };

  return (
    <div className="playlists-page">
      <div className="page-header">
        <h2>📋 播放列表</h2>
        <button className="btn-sm btn-primary" onClick={() => setShowCreate(true)}>+ 新建</button>
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="create-form">
          <input
            className="auth-input"
            placeholder="播放列表名称"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            required
          />
          <input
            className="auth-input"
            placeholder="文件夹路径（如 /media/movies）"
            value={newPath}
            onChange={(e) => setNewPath(e.target.value)}
            required
          />
          <div className="edit-actions">
            <button type="submit" className="btn-sm btn-primary">创建</button>
            <button type="button" className="btn-sm btn-secondary" onClick={() => setShowCreate(false)}>取消</button>
          </div>
        </form>
      )}

      {loading ? (
        <div className="loading">加载中...</div>
      ) : playlists.length === 0 ? (
        <div className="empty-state">
          <p className="empty-icon">📋</p>
          <p>还没有播放列表</p>
          <p className="empty-hint">点击"新建"从文件夹创建播放列表</p>
        </div>
      ) : (
        <div className="playlist-grid">
          {playlists.map((pl) => (
            <div key={pl.id} className="playlist-card">
              <div className="pl-info">
                <h3>{pl.name}</h3>
                <p className="pl-path">{pl.folderPath}</p>
                <span className="pl-mode">{playModeLabels[pl.playMode] || pl.playMode}</span>
              </div>
              <div className="pl-actions">
                <button className="btn-sm btn-primary" onClick={() => handlePlay(pl)}>▶ 播放</button>
                <button className="btn-sm btn-secondary" onClick={() => handleDelete(pl.id)}>删除</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
