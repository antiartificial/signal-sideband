import { useEffect, useRef, useState, useCallback } from 'react'
import mermaid from 'mermaid'
import { useQuery } from '@tanstack/react-query'
import { getVersion } from '../lib/api.ts'
import Card from '../components/Card.tsx'

mermaid.initialize({
  startOnLoad: false,
  theme: 'neutral',
  themeVariables: {
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
    fontSize: '13px',
  },
})

const architectureDiagram = `
graph TB
  subgraph Signal["Signal Protocol"]
    SC[signal-cli<br/>Docker container]
  end

  subgraph Backend["Go Backend"]
    WS[WebSocket Listener]
    API[REST API Server]
    MW[Media Worker]
    AW[AI Vision Worker]
    LPW[Link Preview Worker]
    RW[Reaper Worker]
    DG[Digest Scheduler]
    CW[Cerebro Worker]
  end

  subgraph Storage["PostgreSQL + pgvector"]
    MSG[(messages)]
    ATT[(attachments)]
    URL[(urls)]
    CON[(contacts)]
    DIG[(digests)]
    KG[(cerebro<br/>knowledge graph)]
  end

  subgraph AI["AI Providers"]
    OAI[OpenAI<br/>Embeddings]
    XAI[xAI / Grok<br/>LLM + Vision]
    PPX[Perplexity<br/>Enrichment]
    GEM[Gemini<br/>Image Gen]
  end

  subgraph Frontend["React Frontend"]
    UI[Vite + TypeScript + Tailwind]
  end

  SC -->|WebSocket| WS
  WS -->|Store + Embed| MSG
  WS --> ATT
  WS --> URL
  WS --> CON
  MW -->|Download media| SC
  MW --> ATT
  AW -->|Vision analysis| XAI
  AW --> ATT
  LPW --> URL
  RW -->|Delete expired| MSG
  DG -->|Generate digests| DIG
  CW -->|Extract concepts| KG
  CW -->|Enrich| PPX
  CW -->|Enrich| XAI
  WS -->|Embed text| OAI
  DG -->|Summarize| XAI
  UI -->|REST API| API
  API --> MSG
  API --> ATT
  API --> DIG
  API --> KG
  API --> CON
`

const dataFlowDiagram = `
sequenceDiagram
  participant S as Signal Group
  participant SC as signal-cli
  participant WS as WebSocket Listener
  participant DB as PostgreSQL
  participant OAI as OpenAI
  participant W as Background Workers
  participant UI as React UI

  S->>SC: Message sent in group
  SC->>WS: WebSocket event
  WS->>OAI: Generate embedding
  OAI-->>WS: Vector [1536 dims]
  WS->>DB: Store message + embedding
  WS->>DB: Save attachments metadata
  WS->>DB: Extract & save URLs

  par Background Processing
    W->>SC: Download attachment files
    W->>DB: Update local paths
    W->>DB: AI vision analysis
    W->>DB: Fetch link previews
  end

  UI->>DB: Search / browse via API
  Note over UI: Full-text, semantic,<br/>or filter by sender/date
`

const privacyDiagram = `
graph LR
  subgraph Incoming["All Signal Events"]
    DM[Direct Messages]
    OG[Other Groups]
    TG[Target Group]
  end

  subgraph Filter["FILTER_GROUP_ID"]
    F{Group<br/>matches?}
  end

  subgraph Actions["Processing"]
    STORE[Store + Index]
    DROP[Silently Drop]
    REAP[Reaper<br/>every 60s]
    RDEL[Remote Delete<br/>immediate]
  end

  DM --> F
  OG --> F
  TG --> F
  F -->|No| DROP
  F -->|Yes| STORE
  STORE -->|expires_at reached| REAP
  STORE -->|"Delete for Everyone"| RDEL

  style DROP fill:#fee,stroke:#c00
  style STORE fill:#efe,stroke:#0a0
  style REAP fill:#fef,stroke:#a0a
  style RDEL fill:#fef,stroke:#a0a
`

function MermaidChart({ chart, id }: { chart: string; id: string }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const svgRef = useRef<HTMLDivElement>(null)
  const [scale, setScale] = useState(1)
  const [translate, setTranslate] = useState({ x: 0, y: 0 })
  const dragRef = useRef<{ startX: number; startY: number; startTx: number; startTy: number } | null>(null)

  useEffect(() => {
    if (!svgRef.current) return
    svgRef.current.innerHTML = ''
    setScale(1)
    setTranslate({ x: 0, y: 0 })
    mermaid.render(id, chart).then(({ svg }) => {
      if (svgRef.current) svgRef.current.innerHTML = svg
    })
  }, [chart, id])

  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    const delta = e.deltaY > 0 ? 0.9 : 1.1
    setScale(s => Math.min(Math.max(s * delta, 0.3), 3))
  }, [])

  const handlePointerDown = useCallback((e: React.PointerEvent) => {
    (e.target as HTMLElement).setPointerCapture(e.pointerId)
    dragRef.current = { startX: e.clientX, startY: e.clientY, startTx: translate.x, startTy: translate.y }
  }, [translate])

  const handlePointerMove = useCallback((e: React.PointerEvent) => {
    if (!dragRef.current) return
    setTranslate({
      x: dragRef.current.startTx + (e.clientX - dragRef.current.startX),
      y: dragRef.current.startTy + (e.clientY - dragRef.current.startY),
    })
  }, [])

  const handlePointerUp = useCallback(() => {
    dragRef.current = null
  }, [])

  const resetView = useCallback(() => {
    setScale(1)
    setTranslate({ x: 0, y: 0 })
  }, [])

  return (
    <div className="relative">
      <div className="absolute top-2 right-2 flex gap-1 z-10">
        <button onClick={() => setScale(s => Math.min(s * 1.2, 3))} className="px-2 py-1 text-xs bg-apple-accent-dim rounded hover:bg-apple-border transition-colors">+</button>
        <button onClick={() => setScale(s => Math.max(s * 0.8, 0.3))} className="px-2 py-1 text-xs bg-apple-accent-dim rounded hover:bg-apple-border transition-colors">&minus;</button>
        <button onClick={resetView} className="px-2 py-1 text-xs bg-apple-accent-dim rounded hover:bg-apple-border transition-colors">Reset</button>
      </div>
      <div
        ref={containerRef}
        className="overflow-hidden cursor-grab active:cursor-grabbing py-4"
        style={{ touchAction: 'none' }}
        onWheel={handleWheel}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
      >
        <div
          ref={svgRef}
          className="flex justify-center"
          style={{ transform: `translate(${translate.x}px, ${translate.y}px) scale(${scale})`, transformOrigin: 'center center' }}
        />
      </div>
    </div>
  )
}

export default function About() {
  const { data: versionInfo } = useQuery({
    queryKey: ['version'],
    queryFn: getVersion,
    staleTime: Infinity,
  })

  return (
    <div>
      <h2 className="text-2xl font-semibold tracking-tight mb-2"><i className="fawsb fa-circle-info text-apple-secondary mr-2" />About Signal Sideband</h2>
      <p className="text-sm text-apple-secondary mb-8">
        A Signal intelligence dashboard — captures messages from a Signal group via signal-cli,
        stores them with vector embeddings, and provides search, digests, media gallery,
        knowledge graph, and contact management.
      </p>

      {/* Architecture */}
      <h3 className="text-lg font-medium mb-3">
        <i className="fawsb fa-share-nodes text-apple-secondary mr-2" />
        Architecture
      </h3>
      <Card className="p-4 mb-8">
        <MermaidChart chart={architectureDiagram} id="arch" />
      </Card>

      {/* Data Flow */}
      <h3 className="text-lg font-medium mb-3">
        <i className="fawsb fa-arrow-progress text-apple-secondary mr-2" />
        Data Flow
      </h3>
      <Card className="p-4 mb-8">
        <MermaidChart chart={dataFlowDiagram} id="flow" />
      </Card>

      {/* Privacy Model */}
      <h3 className="text-lg font-medium mb-3">
        <i className="fawsb fa-shield-check text-apple-secondary mr-2" />
        Privacy Model
      </h3>
      <Card className="p-4 mb-8">
        <MermaidChart chart={privacyDiagram} id="privacy" />
      </Card>

      {/* Features */}
      <h3 className="text-lg font-medium mb-3">
        <i className="fawsb fa-stars text-apple-secondary mr-2" />
        What It Does
      </h3>
      <div className="grid gap-4 sm:grid-cols-2 mb-8">
        {[
          { icon: 'fa-magnifying-glass', title: 'Semantic Search', desc: 'Find any conversation with natural language. Powered by OpenAI embeddings and pgvector.' },
          { icon: 'fa-newspaper', title: 'AI Digests', desc: 'Daily summaries in multiple styles — standard newsletter, Lord of the Rings chronicle, Confucian wisdom, or South Park.' },
          { icon: 'fa-share-nodes', title: 'Knowledge Graph', desc: 'Cerebro extracts concepts and relationships from conversations, enriched by Perplexity and Grok.' },
          { icon: 'fa-images', title: 'Media Gallery', desc: 'All shared images with AI vision analysis. Search photos by what\'s in them.' },
          { icon: 'fa-link', title: 'Link Collection', desc: 'Every URL shared in chat, with automatic link previews. Never lose that article again.' },
          { icon: 'fa-shield-check', title: 'Privacy Respecting', desc: 'Group filtering, disappearing messages honored, remote deletes respected. Read-only — never sends back to Signal.' },
        ].map(f => (
          <Card key={f.title} className="p-5">
            <div className="flex items-center gap-2 mb-2">
              <i className={`fawsb ${f.icon} text-apple-blue`} />
              <h4 className="font-medium">{f.title}</h4>
            </div>
            <p className="text-sm text-apple-secondary leading-relaxed">{f.desc}</p>
          </Card>
        ))}
      </div>

      {/* Version */}
      <div className="text-xs text-apple-secondary text-center font-mono">
        Signal Sideband {versionInfo?.version ?? '...'}
      </div>
    </div>
  )
}
