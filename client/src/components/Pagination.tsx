import { useState } from 'react';
import './Pagination.css';

interface PaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

export default function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  const [jumpInput, setJumpInput] = useState('');

  if (totalPages <= 1) return null;

  const getPageNumbers = (): (number | 'ellipsis')[] => {
    const pages: (number | 'ellipsis')[] = [];
    const windowSize = 2; // show ±2 pages around current

    const windowStart = Math.max(1, page - windowSize);
    const windowEnd = Math.min(totalPages, page + windowSize);

    // First page (with gap detection)
    if (windowStart > 1) {
      pages.push(1);
      if (windowStart > 2) {
        pages.push('ellipsis');
      }
      // If windowStart === 2, page 2 enters the window below — no ellipsis needed
    }

    // Window pages
    for (let i = windowStart; i <= windowEnd; i++) {
      pages.push(i);
    }

    // Last page (with gap detection)
    if (windowEnd < totalPages) {
      if (windowEnd < totalPages - 1) {
        pages.push('ellipsis');
      }
      pages.push(totalPages);
    }

    return pages;
  };

  const handleJump = () => {
    const target = parseInt(jumpInput, 10);
    if (isNaN(target) || target < 1 || target > totalPages) return;
    onPageChange(target);
    setJumpInput('');
  };

  const handleJumpKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleJump();
  };

  const pages = getPageNumbers();

  return (
    <div className="pagination">
      <button
        className="page-btn page-nav"
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
      >
        上一页
      </button>

      {pages.map((p, i) =>
        p === 'ellipsis' ? (
          <span key={`e-${i}`} className="page-ellipsis">…</span>
        ) : (
          <button
            key={p}
            className={`page-btn page-num ${p === page ? 'active' : ''}`}
            onClick={() => onPageChange(p)}
          >
            {p}
          </button>
        )
      )}

      <button
        className="page-btn page-nav"
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
      >
        下一页
      </button>

      <div className="page-jump">
        <input
          className="jump-input"
          type="text"
          inputMode="numeric"
          placeholder="页码"
          value={jumpInput}
          onChange={(e) => setJumpInput(e.target.value.replace(/\D/g, ''))}
          onKeyDown={handleJumpKeyDown}
        />
        <button className="page-btn jump-btn" onClick={handleJump}>
          跳转
        </button>
      </div>
    </div>
  );
}
