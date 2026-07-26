import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { mediaAPI } from '../api/client';
import { useStore } from '../stores';
import './Browse.css';

interface MediaItem {
  id: number;
  title: string;
  type: string;
  duration: number;
  coverPath: string;
  resolution: string;
  tags: { id: number; name: string; color: string }[];
  createdAt: string;
}

export default function Browse() {
  const [items, setItems] = useState<MediaItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const filter = useStore((s) => s.filter);


  const fetchMedia = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {
        page: String(page),
        pageSize: '50',
        sort: filter.sortBy,
        order: filter.order,
      };
      if (filter.typeFilter) params.type = filter.typeFilter;

      const res = await mediaAPI.list(params);
      setItems(res.data.items);
      setTotalPages(res.data.totalPages);
    } catch (err) {
      console.error('Failed to load media:', err);
    } finally {
      setLoading(false);
    }
  }, [page, filter]);

  useEffect(() => { fetchMedia(); }, [fetchMedia]);

  const formatDuration = (s: number): string => {
    if (!s) return '';
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const sec = Math.floor(s % 60);
    if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`;
    return `${m}:${String(sec).padStart(2, '0')}`;
  };

  return (
    <div className="browse">
      {loading ? (
        <div className="loading">加载中...</div>
      ) : items.length === 0 ? (
        <div className="empty-state">
          <p className="empty-icon">📁</p>
          <p>还没有媒体文件</p>
          <p className="empty-hint">请联系管理员添加扫描文件夹</p>
        </div>
      ) : (
        <>
          <div className="media-grid">
            {items.map((item) => (
              <Link
                to={`/media/${item.id}`}
                key={item.id}
                className="media-card focusable"
              >
                <div className="card-cover">
                  {item.coverPath ? (
                    <img
                      src={`/api/stream/cover/${item.id}?token=${localStorage.getItem('token') || ''}`}
                      alt={item.title}
                      loading="lazy"
                    />
                  ) : (
                    <div className="card-placeholder">
                      {item.type === 'audio' ? '🎵' : '🎬'}
                    </div>
                  )}
                  <span className="card-duration">{formatDuration(item.duration)}</span>
                  {item.type === 'audio' && <span className="card-type-icon">🎵</span>}
                </div>
                <div className="card-info">
                  <h3 className="card-title">{item.title}</h3>
                  {item.resolution && <span className="card-res">{item.resolution}</span>}
                  {item.tags.length > 0 && (
                    <div className="card-tags">
                      {item.tags.slice(0, 3).map((tag) => (
                        <span key={tag.id} className="tag-badge" style={{ background: tag.color }}>
                          {tag.name}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </Link>
            ))}
          </div>

          {totalPages > 1 && (
            <div className="pagination">
              <button
                className="page-btn"
                disabled={page <= 1}
                onClick={() => setPage(page - 1)}
              >
                上一页
              </button>
              <span className="page-info">{page} / {totalPages}</span>
              <button
                className="page-btn"
                disabled={page >= totalPages}
                onClick={() => setPage(page + 1)}
              >
                下一页
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
