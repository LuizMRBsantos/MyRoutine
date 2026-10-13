// Ícones do app: uma família só, traço de 1,6 px no estilo Phosphor,
// desenhados à mão (sem biblioteca). Herdam a cor do texto (currentColor).
export type IconName =
  | 'home' | 'journal' | 'habits' | 'planner' | 'finance' | 'health'
  | 'studies' | 'review' | 'account' | 'invites' | 'more' | 'logout'

const PATHS: Record<IconName, React.ReactNode> = {
  home: <path d="M4 11l8-6 8 6v8a1 1 0 0 1-1 1h-4v-6H9v6H5a1 1 0 0 1-1-1z" />,
  journal: <><path d="M6 4h10a2 2 0 0 1 2 2v14H8a2 2 0 0 1-2-2z" /><path d="M6 18a2 2 0 0 1 2-2h10" /><path d="M10 8h5" /></>,
  habits: <><circle cx="12" cy="12" r="8" /><path d="M8.5 12.5l2.3 2.3 4.7-5" /></>,
  planner: <><rect x="4" y="5" width="16" height="15" rx="2" /><path d="M4 10h16M9 3v4M15 3v4" /></>,
  finance: <><rect x="3.5" y="6" width="17" height="13" rx="2" /><path d="M3.5 10h17M15.5 14.5h2" /></>,
  health: <path d="M3 12h4l2-4 3 8 2-4h7" />,
  studies: <><path d="M3 8l9-4 9 4-9 4z" /><path d="M7 10v5c0 1.5 2.2 3 5 3s5-1.5 5-3v-5" /></>,
  review: <><path d="M4 12a8 8 0 1 0 2.4-5.7" /><path d="M4 4v4h4" /></>,
  account: <><circle cx="12" cy="9" r="3.5" /><path d="M5 20c1.2-3.5 4-5 7-5s5.8 1.5 7 5" /></>,
  invites: <><rect x="3.5" y="6" width="17" height="12" rx="2" /><path d="M4 7l8 6 8-6" /></>,
  more: <><circle cx="6" cy="12" r="1.2" /><circle cx="12" cy="12" r="1.2" /><circle cx="18" cy="12" r="1.2" /></>,
  logout: <><path d="M14 4h4a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-4" /><path d="M10 8l-4 4 4 4M6 12h10" /></>,
}

export function Icon({ name, size = 20 }: { name: IconName; size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth={1.6} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {PATHS[name]}
    </svg>
  )
}
