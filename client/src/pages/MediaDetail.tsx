import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { mediaAPI, searchAPI } from '../api/client';
import { useStore } from '../stores';
import './MediaDetail.css';

interface MediaInfo {
  id: number;
  title: string;
  description: string;
  type: string;
  path: string;
  duration: number;
  resolution: string;
  cover_path: string;
  file_size: number;
  codec: string;
  bitrate: number;
  tags: { id: number; name: string; color: string }[];
  created_at: string;
  updated_at: string;
}

export default function MediaDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { setCurrentMedia } = useStore();

  const [media, setMedia] = useState<MediaInfo | null>(null);
  const [allTags, setAllTags] = useState<{ id: number; name: string; color: string }[]>([]);
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    Promise.all([
      mediaAPI.get(Number(id)),
      searchAPI.getTags(),
    ]).then(([mediaRes, tagsRes]) => {
      const m = mediaRes.data;
      setMedia(m);
      setTitle(m.title);
      setDescription(m.description);
      setAllTags(tagsRes.data.items);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, [id]);

  const handlePlay = () => {
    if (media) {
      setCurrentMedia({ id: media.id, title: media.title, type: media.type, cover_path: media.cover_path });
      navigate('/player');
    }
  };

  const handleSave = async () => {
    if (!media) return;
    try {
      await mediaAPI.update(media.id, {
        title,
        description,
        tagIds: media.tags.map((t) => t.id),
      });
      setEditing(false);
      setMedia({ ...media, title, description, tags: media.tags });
    } catch (err) {
      console.error('Save failed:', err);
    }
  };

  const toggleTag = async (tagId: number) => {
    if (!media) return;
    const hasTag = media.tags.find((t) => t.id === tagId);
    let newTags: number[];
    if (hasTag) {
      newTags = media.tags.filter((t) => t.id !== tagId).map((t) => t.id);
      setMedia({ ...media, tags: media.tags.filter((t) => t.id !== tagId) });
    } else {
      const tag = allTags.find((t) => t.id === tagId);
      newTags = [...media.tags.map((t) => t.id), tagId];
      if (tag) setMedia({ ...media, tags: [...media.tags, tag] });
    }
    await mediaAPI.update(media.id, { tagIds: newTags });
  };

  const handleCoverUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !media) return;
    try {
      await mediaAPI.uploadCover(media.id, file);
      // Refresh
      const res = await mediaAPI.get(media.id);
      setMedia(res.data);
    } catch (err) {
      console.error('Cover upload failed:', err);
    }
  };

  const formatSize = (bytes: number): string => {
    if (bytes > 1024 * 1024 * 1024) return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
    if (bytes > 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(0) + ' MB';
    return (bytes / 1024).toFixed(0) + ' KB';
  };

  const formatDuration = (s: number): string => {
    if (!s) return '';
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const sec = Math.floor(s % 60);
    if (h > 0) return `${h}小时${m}分${sec}秒`;
    return `${m}分${sec}秒`;
  };

  if (loading) return <div className="loading">加载中...</div>;
  if (!media) return <div className="loading">媒体不存在</div>;

  return (
    <div className="media-detail">
      <div className="detail-hero">
        <div className="detail-cover">
          {media.cover_path ? (
            <img src={`/api/stream/cover/${media.id}`} alt={media.title} />
          ) : (
            <div className="card-placeholder">{media.type === 'audio' ? '🎵' : '🎬'}</div>
          )}
          <label className="cover-upload-btn">
            📷 更换封面
            <input type="file" accept="image/jpeg,image/png" onChange={handleCoverUpload} hidden />
          </label>
        </div>

        <div className="detail-info">
          {editing ? (
            <>
              <input className="edit-input" value={title} onChange={(e) => setTitle(e.target.value)} />
              <textarea className="edit-textarea" value={description} onChange={(e) => setDescription(e.target.value)} rows={3} />
              <div className="edit-actions">
                <button className="btn-sm btn-primary" onClick={handleSave}>保存</button>
                <button className="btn-sm btn-secondary" onClick={() => setEditing(false)}>取消</button>
              </div>
            </>
          ) : (
            <>
              <h1 className="detail-title">{media.title}</h1>
              {media.description && <p className="detail-desc">{media.description}</p>}
              <button className="btn-sm btn-secondary" onClick={() => setEditing(true)}>✏️ 编辑</button>
            </>
          )}

          <div className="detail-meta">
            <span>{media.type === 'video' ? '🎬 视频' : '🎵 音频'}</span>
            <span>{formatDuration(media.duration)}</span>
            {media.resolution && <span>{media.resolution}</span>}
            <span>{formatSize(media.file_size)}</span>
            {media.codec && <span>{media.codec}</span>}
          </div>

          <div className="detail-tags">
            <span className="tags-label">标签：</span>
            {media.tags.map((tag) => (
              <span
                key={tag.id}
                className="tag-badge tag-removable"
                style={{ background: tag.color }}
                onClick={() => toggleTag(tag.id)}
              >
                {tag.name} ✕
              </span>
            ))}
            {/* Available tags to add */}
            {allTags.filter((t) => !media.tags.find((mt) => mt.id === t.id)).slice(0, 10).map((tag) => (
              <span
                key={tag.id}
                className="tag-badge tag-addable"
                style={{ border: `1px solid ${tag.color}`, color: tag.color }}
                onClick={() => toggleTag(tag.id)}
              >
                + {tag.name}
              </span>
            ))}
          </div>

          <button className="btn-play" onClick={handlePlay}>
            ▶ {media.type === 'video' ? '播放视频' : '播放音频'}
          </button>
        </div>
      </div>
    </div>
  );
}
