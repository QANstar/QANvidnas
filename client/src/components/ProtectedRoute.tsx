import { Navigate } from 'react-router-dom';
import { useStore } from '../stores';

export default function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = useStore((s) => s.token);

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}
