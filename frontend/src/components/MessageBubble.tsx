type Props = {
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
};

export default function MessageBubble({ role, content, timestamp }: Props) {
  const isUser = role === 'user';
  return (
    <div style={{ display: 'flex', justifyContent: isUser ? 'flex-end' : 'flex-start', marginBottom: 10 }}>
      <div
        style={{
          maxWidth: '78%',
          padding: '10px 12px',
          borderRadius: 12,
          background: isUser ? '#2563eb' : '#eceff3',
          color: isUser ? '#ffffff' : '#1f2937'
        }}
      >
        <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.4 }}>{content}</div>
        <div style={{ marginTop: 6, fontSize: 11, opacity: 0.8, textAlign: isUser ? 'right' : 'left' }}>{timestamp}</div>
      </div>
    </div>
  );
}
