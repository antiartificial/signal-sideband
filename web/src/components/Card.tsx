import type { CSSProperties, ReactNode } from 'react'

export default function Card({
  children,
  className = '',
  onClick,
  style,
}: {
  children: ReactNode
  className?: string
  onClick?: () => void
  style?: CSSProperties
}) {
  return (
    <div
      onClick={onClick}
      style={style}
      className={`bg-apple-card rounded-xl border border-apple-border shadow-sm card-glow
        animate-fade-in-up transition-all duration-300 ease-out
        hover:translate-y-[-1px] hover:shadow-md ${
        onClick ? 'cursor-pointer' : ''
      } ${className}`}
    >
      {children}
    </div>
  )
}
