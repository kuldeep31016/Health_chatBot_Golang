import { useEffect, useMemo, useRef, useState } from 'react';
import { sendMessage, ConnectionState } from '../api/chat';
import InputBox from './InputBox';
import MessageBubble from './MessageBubble';

type Message = {
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
};

type Props = {
  onConnectionStateChange: (state: ConnectionState) => void;
};

export default function ChatWindow({ onConnectionStateChange }: Props) {
  const [messages, setMessages] = useState<Message[]>([
    {
      role: 'assistant',
      content: 'Hi! I\'m your AI Health Assistant. Tell me how you\'re feeling, and I\'ll help using your profile data.',
      timestamp: new Date().toLocaleTimeString()
    }
  ]);
  const [loading, setLoading] = useState(false);
  const endRef = useRef<HTMLDivElement | null>(null);

  const sessionId = useMemo(() => 'session-' + Math.random().toString(36).slice(2, 10), []);

  const scrollToBottom = () => {
    requestAnimationFrame(() => endRef.current?.scrollIntoView({ behavior: 'smooth' }));
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, loading]);

  const onSend = async (value: string) => {
    const now = new Date().toLocaleTimeString();
    setMessages((prev) => [...prev, { role: 'user', content: value, timestamp: now }]);
    setLoading(true);

    try {
      const response = await sendMessage(value, sessionId, onConnectionStateChange);
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: response, timestamp: new Date().toLocaleTimeString() }
      ]);
    } catch {
      setMessages((prev) => [
        ...prev,
        {
          role: 'assistant',
          content: "I couldn't reply just now due to a temporary issue. Please try again.",
          timestamp: new Date().toLocaleTimeString()
        }
      ]);
      onConnectionStateChange('disconnected');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ flex: 1, overflowY: 'auto', padding: 12, background: '#f8fafc' }}>
        {messages.map((m, idx) => (
          <MessageBubble key={idx} role={m.role} content={m.content} timestamp={m.timestamp} />
        ))}
        {loading && <MessageBubble role="assistant" content="" timestamp="" isTyping />}
        <div ref={endRef} />
      </div>
      <InputBox loading={loading} onSend={onSend} />
    </div>
  );
}
