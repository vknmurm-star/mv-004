export default function MaxButton({ href = '#', label = 'Написать в MAX' }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="btn lg"
      style={{ background: 'linear-gradient(135deg, #6366f1, #22d3ee)', color: '#fff', borderColor: 'transparent' }}
    >
      <span>💬</span> {label}
    </a>
  )
}