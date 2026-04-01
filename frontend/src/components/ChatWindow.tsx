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

const CHAT_HISTORY_KEY = 'health_assistant_chat_history';
const SESSION_ID_KEY = 'health_assistant_session_id';

const getWelcomeMessage = (): Message => ({
  role: 'assistant',
  content: 'Hi! I\'m your AI Health Assistant. Tell me how you\'re feeling, and I\'ll help using your profile data.',
  timestamp: new Date().toLocaleTimeString()
});

export default function ChatWindow({ onConnectionStateChange }: Props) {
  const [messages, setMessages] = useState<Message[]>(() => {
    try {
      const raw = localStorage.getItem(CHAT_HISTORY_KEY);
      if (!raw) {
        return [getWelcomeMessage()];
      }

      const parsed = JSON.parse(raw);
      if (!Array.isArray(parsed) || parsed.length === 0) {
        return [getWelcomeMessage()];
      }

      return parsed as Message[];
    } catch {
      return [getWelcomeMessage()];
    }
  });
  const [loading, setLoading] = useState(false);
  const endRef = useRef<HTMLDivElement | null>(null);

  const sessionId = useMemo(() => {
    const existing = localStorage.getItem(SESSION_ID_KEY);
    if (existing) {
      return existing;
    }

    const generated = 'session-' + Math.random().toString(36).slice(2, 10);
    localStorage.setItem(SESSION_ID_KEY, generated);
    return generated;
  }, []);

  const scrollToBottom = () => {
    requestAnimationFrame(() => endRef.current?.scrollIntoView({ behavior: 'smooth' }));
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, loading]);

  useEffect(() => {
    localStorage.setItem(CHAT_HISTORY_KEY, JSON.stringify(messages));
  }, [messages]);

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
