import { useState } from 'react';
import ChatWindow from './components/ChatWindow';
import ConnectionStatus from './components/ConnectionStatus';
import { ConnectionState } from './api/chat';

export default function App() {
  const [connectionState, setConnectionState] = useState<ConnectionState>('reconnecting');

  return (
    <div
      style={{
        minHeight: '100vh',
        background: 'linear-gradient(180deg, #eef2ff 0%, #f8fafc 100%)',
        padding: 20,
        boxSizing: 'border-box'
      }}
    >
      <div
        style={{
          maxWidth: 860,
          margin: '0 auto',
          height: 'calc(100vh - 40px)',
          background: '#ffffff',
          borderRadius: 16,
          boxShadow: '0 12px 30px rgba(15, 23, 42, 0.08)',
          border: '1px solid #e5e7eb',
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column'
        }}
      >
        <ConnectionStatus overrideStatus={connectionState} />
        <ChatWindow onConnectionStateChange={setConnectionState} />
      </div>
    </div>
  );
}
