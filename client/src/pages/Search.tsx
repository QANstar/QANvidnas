import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { searchAPI } from '../api/client';
import { useStore } from '../stores';
import './Search.css';

interface Tag {
  id: number;
  name: string;
  color: string;
}

interface SearchItem {
  id: number;
  title: string;
  description: string;
  type: string;
  duration: number;
  coverPath: string;
}

export default function Search() {
  const [query, setQuery] = useState('');
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [allTags, setAllTags] = useState<Tag[]>([]);
  const [results, setResults] = useState<SearchItem[]>([]);
  const [loading, setLoading] = useState(false);

  const filter = useStore((s) => s.filter);

  useEffect(() => {
    searchAPI.getTags().then((res) => setAllTags(res.data.items));
  }, []);

  const doSearch = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (query.trim()) params.q = query.trim();
      if (selectedTags.length > 0) params.tags = selectedTags.join(',');
      if (filter.typeFilter) params.type = filter.typeFilter;

      const res = await searchAPI.search(params);
      setResults(res.data.items);
    } catch (err) {
      console.error('Search failed:', err);
    } finally {
      setLoading(false);
    }
  }, [query, selectedTags, filter.typeFilter]);

  useEffect(() => {
    const timer = setTimeout(doSearch, 300);
    return () => clearTimeout(timer);
  }, [doSearch]);

  const toggleTag = (tagName: string) => {
    setSelectedTags((prev) =>
      prev.includes(tagName) ? prev.filter((t) => t !== tagName) : [...prev, tagName]
    );
  };

  const formatDuration = (s: number): string => {
    if (!s) return '';
    const m = Math.floor(s / 60);
    const sec = Math.floor(s % 60);
    return `${m}:${String(sec).padStart(2, '0')}`;
  };

  return (
    <div className="search-page">
      <div className="search-bar-wrapper">
        <input
          className="search-input"
          type="text"
          placeholder="搜索媒体名称、标签、描述..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          autoFocus
        />
        <span className="search-icon">🔍</span>
      </div>

      <div className="tag-filters">
        {allTags.map((tag) => (
          <button
            key={tag.id}
            className={`tag-filter-btn ${selectedTags.includes(tag.name) ? 'active' : ''}`}
            style={{
              borderColor: tag.color,
              color: selectedTags.includes(tag.name) ? 'white' : tag.color,
              background: selectedTags.includes(tag.name) ? tag.color : 'transparent',
            }}
            onClick={() => toggleTag(tag.name)}
          >
            {tag.name}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="loading">搜索中...</div>
      ) : results.length === 0 ? (
        <div className="empty-state">
          <p className="empty-icon">🔍</p>
          <p>{query ? '未找到相关内容' : '输入关键词或选择标签开始搜索'}</p>
        </div>
      ) : (
        <>
          <p className="result-count">找到 {results.length} 个结果</p>
          <div className="search-results">
            {results.map((item) => (
              <Link to={`/media/${item.id}`} key={item.id} className="search-result-item">
                <div className="sr-cover">
                  {item.coverPath ? (
                    <img src={`/api/stream/cover/${item.id}?token=${localStorage.getItem('token') || ''}`} alt={item.title} />
                  ) : (
                    <div className="card-placeholder">{item.type === 'audio' ? '🎵' : '🎬'}</div>
                  )}
                </div>
                <div className="sr-info">
                  <h3 className="sr-title">{item.title}</h3>
                  {item.description && <p className="sr-desc">{item.description}</p>}
                  <span className="sr-meta">
                    {item.type === 'video' ? '🎬' : '🎵'} {formatDuration(item.duration)}
                  </span>
                </div>
              </Link>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
