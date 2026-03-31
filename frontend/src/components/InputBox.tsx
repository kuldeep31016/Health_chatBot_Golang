import { FormEvent, useState } from 'react';

type Props = {
  loading: boolean;
  onSend: (value: string) => Promise<void>;
};

export default function InputBox({ loading, onSend }: Props) {
  const [text, setText] = useState('');

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const value = text.trim();
    if (!value || loading) return;
    setText('');
    await onSend(value);
  };

  return (
    <form onSubmit={submit} style={{ display: 'flex', gap: 10, padding: 12, borderTop: '1px solid #e5e7eb', background: '#fff' }}>
      <input
        value={text}
        disabled={loading}
        onChange={(e) => setText(e.target.value)}
        placeholder={loading ? 'Assistant is typing...' : 'Ask about your health...'}
        style={{ flex: 1, border: '1px solid #d1d5db', borderRadius: 10, padding: '10px 12px', outline: 'none' }}
      />
      <button
        type="submit"
        disabled={loading || !text.trim()}
        style={{
          border: 'none',
          borderRadius: 10,
          padding: '10px 14px',
          background: loading ? '#9ca3af' : '#111827',
          color: '#fff',
          cursor: loading ? 'not-allowed' : 'pointer'
        }}
      >
        {loading ? '...' : 'Send'}
      </button>
    </form>
  );
}
