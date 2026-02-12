export default function Badge({ children, variant = 'default' }: { children: React.ReactNode; variant?: 'default' | 'blue' | 'green' }) {
  const styles = {
    default: 'bg-gray-100 text-apple-secondary',
    blue: 'bg-blue-50 text-apple-blue',
    green: 'bg-green-50 text-green-700',
  }

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium ${styles[variant]}`}>
      {children}
    </span>
  )
}
