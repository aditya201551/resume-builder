import { NavLink, Outlet, useLocation } from 'react-router'
import {
  BriefcaseBusiness,
  ChevronDown,
  Files,
  House,
  LogOut,
} from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import Brand from '@/components/layout/Brand'
import { cn } from '@/lib/utils'

export default function AppShell() {
  const { pathname } = useLocation()
  const { user, logout } = useAuth()
  const name = user?.name || user?.email || 'Your workspace'
  const initials = name
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join('')
    .toUpperCase()

  return (
    <div className={cn('app-shell', pathname === '/resumes' && 'app-shell-home')}>
      <header className="app-sidebar">
        <NavLink
          to="/resumes"
          className="brand-link"
          aria-label="Resume Builder home"
        >
          <Brand />
        </NavLink>
        <nav aria-label="Main navigation" className="app-navigation">
          {[
            { to: '/resumes', label: 'Home', icon: House },
            { to: '/resumes/library', label: 'My resumes', icon: Files },
            { to: '/jobs', label: 'Jobs', icon: BriefcaseBusiness },
          ].map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              end={to !== '/jobs'}
              className={({ isActive }) =>
                cn('app-nav-link', isActive && 'is-active')
              }
            >
              <Icon className="size-[18px]" aria-hidden /> {label}
            </NavLink>
          ))}
        </nav>
        <div className="app-account">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              className="profile-button"
              aria-label="Account menu"
            >
              <Avatar className="size-9">
                <AvatarFallback className="bg-secondary text-secondary-foreground text-xs">
                  {initials}
                </AvatarFallback>
              </Avatar>
              <span className="min-w-0 flex-1 truncate text-left text-sm font-medium">
                {name}
              </span>
              <ChevronDown
                className="size-3.5 text-muted-foreground"
                aria-hidden
              />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="start"
            side="bottom"
            className="max-w-72 rounded-lg p-2"
          >
            {user && (
              <div className="truncate px-2 py-2 text-xs text-muted-foreground">
                {user.email}
              </div>
            )}
            <DropdownMenuItem onSelect={() => logout()}>
              <LogOut className="size-4" /> Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        </div>
      </header>
      <main className="app-main">
        <Outlet />
      </main>
    </div>
  )
}
