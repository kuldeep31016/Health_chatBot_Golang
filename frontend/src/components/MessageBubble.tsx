type Props = {
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
  isTyping?: boolean;
};

export default function MessageBubble({ role, content, timestamp, isTyping = false }: Props) {
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
        {isTyping ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, minHeight: 18 }} aria-label="Assistant is typing">
            {[0, 1, 2].map((index) => (
              <span
                key={index}
                style={{
                  width: 7,
                  height: 7,
                  borderRadius: '50%',
                  background: '#6b7280',
                  opacity: 0.45,
                  animation: `typingPulse 1s ${index * 0.18}s infinite ease-in-out`
                }}
              />
            ))}
            <style>{`@keyframes typingPulse { 0%, 80%, 100% { transform: scale(0.8); opacity: 0.35; } 40% { transform: scale(1); opacity: 1; } }`}</style>
          </div>
        ) : (
          <>
            <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.4 }}>{content}</div>
            <div style={{ marginTop: 6, fontSize: 11, opacity: 0.8, textAlign: isUser ? 'right' : 'left' }}>{timestamp}</div>
          </>
        )}
      </div>
    </div>
  );
}
