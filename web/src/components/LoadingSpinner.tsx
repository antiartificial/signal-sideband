export default function LoadingSpinner({ message = 'Loading...' }: { message?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-apple-secondary">
      <div className="w-8 h-8 border-2 border-apple-border border-t-apple-blue rounded-full animate-spin mb-3" />
      <p className="text-sm">{message}</p>
    </div>
  )
}
