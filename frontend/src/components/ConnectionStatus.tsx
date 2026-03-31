import { useEffect, useState } from 'react';
import { checkHealth, ConnectionState } from '../api/chat';

type Props = {
  overrideStatus?: ConnectionState;
};

export default function ConnectionStatus({ overrideStatus }: Props) {
  const [status, setStatus] = useState<ConnectionState>('reconnecting');

  useEffect(() => {
    let mounted = true;

    const poll = async () => {
      const healthy = await checkHealth();
      if (!mounted) return;
      setStatus((current) => {
        if (overrideStatus && overrideStatus !== 'connected') return overrideStatus;
        return healthy ? 'connected' : current === 'connected' ? 'reconnecting' : 'disconnected';
      });
    };

    poll();
    const id = window.setInterval(poll, 5000);
    return () => {
      mounted = false;
      window.clearInterval(id);
    };
  }, [overrideStatus]);

  const effective = overrideStatus && overrideStatus !== 'connected' ? overrideStatus : status;

  const config = {
    connected: { color: '#1db954', label: 'Connected' },
    reconnecting: { color: '#f5b301', label: 'Reconnecting...' },
    disconnected: { color: '#e74c3c', label: 'Disconnected' }
  }[effective];

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', background: '#f8f9fb', borderBottom: '1px solid #e5e7eb' }}>
      <span style={{ width: 10, height: 10, borderRadius: '50%', background: config.color, display: 'inline-block' }} />
      <span style={{ fontSize: 13, color: '#374151' }}>{config.label}</span>
    </div>
  );
}
