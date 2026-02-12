export default function Pagination({
  total,
  limit,
  offset,
  onChange,
}: {
  total: number
  limit: number
  offset: number
  onChange: (offset: number) => void
}) {
  const totalPages = Math.ceil(total / limit)
  const currentPage = Math.floor(offset / limit) + 1

  if (totalPages <= 1) return null

  return (
    <div className="flex items-center justify-between pt-4">
      <p className="text-sm text-apple-secondary">
        {offset + 1}–{Math.min(offset + limit, total)} of {total}
      </p>
      <div className="flex gap-1.5">
        <button
          onClick={() => onChange(Math.max(0, offset - limit))}
          disabled={currentPage === 1}
          className="px-3 py-1.5 text-sm rounded-lg border border-apple-border disabled:opacity-40 hover:bg-gray-50 transition-colors"
        >
          Previous
        </button>
        <button
          onClick={() => onChange(offset + limit)}
          disabled={currentPage === totalPages}
          className="px-3 py-1.5 text-sm rounded-lg border border-apple-border disabled:opacity-40 hover:bg-gray-50 transition-colors"
        >
          Next
        </button>
      </div>
    </div>
  )
}
