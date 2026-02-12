export default function EmptyState({ title, description }: { title: string; description?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <div className="w-16 h-16 bg-gray-100 rounded-2xl flex items-center justify-center mb-4">
        <span className="text-2xl text-apple-secondary">∅</span>
      </div>
      <h3 className="text-base font-medium text-apple-text mb-1">{title}</h3>
      {description && <p className="text-sm text-apple-secondary max-w-sm">{description}</p>}
    </div>
  )
}
