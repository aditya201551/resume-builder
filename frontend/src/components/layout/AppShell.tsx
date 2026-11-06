import { useEffect, useRef, useState, type CSSProperties } from 'react'
import { Link, matchPath, Outlet, useLocation } from 'react-router'
import { ChevronUp, LogOut, X } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Sidebar, SidebarContent, SidebarFooter, SidebarHeader, SidebarMenu,
  SidebarMenuButton, SidebarMenuItem, SidebarProvider, SidebarTrigger, useSidebar,
} from '@/components/ui/sidebar'
import { Sheet, SheetContent, SheetTitle } from '@/components/ui/sheet'
import { TooltipProvider } from '@/components/ui/tooltip'
import Brand from '@/components/layout/Brand'
import { visibleNavigation } from '@/app/navigation'
import { useFeatures } from '@/platform/features/useFeatures'
import { cn } from '@/lib/utils'

const COLLAPSED_KEY = 'resume-builder.navigation.collapsed'

function AppSidebarContent({ overlay = false, close }: { overlay?: boolean; close: () => void }) {
  const { pathname } = useLocation()
  const { enabled } = useFeatures()
  const { user, logout } = useAuth()
  const { state, isMobile } = useSidebar()
  const editor = Boolean(matchPath('/resumes/:id/edit', pathname))
  const compact = state === 'collapsed' && !isMobile && !overlay
  const name = user?.name || user?.email || 'Your workspace'
  const initials = name.split(' ').filter(Boolean).slice(0, 2).map((part) => part[0]).join('').toUpperCase()

  return (
    <>
      <SidebarHeader className="app-sidebar-heading">
        <Link to="/resumes" className="brand-link" aria-label="Resume Builder home" onClick={close}>
          <Brand />
        </Link>
        {overlay || isMobile ? (
          <Button variant="ghost" size="icon" className="sidebar-toggle" aria-label="Close navigation" onClick={close}>
            <X size={19} aria-hidden />
          </Button>
        ) : <SidebarTrigger className="sidebar-toggle" title={compact ? 'Expand sidebar' : 'Collapse sidebar'} />}
      </SidebarHeader>
      <SidebarContent>
        <nav aria-label="Main navigation" className="app-navigation">
          <SidebarMenu>
            {visibleNavigation(enabled).map(({ to, label, icon: Icon, end }) => {
              const active = Boolean(matchPath({ path: to, end }, pathname)) || (editor && to === '/resumes/library')
              return (
                <SidebarMenuItem key={to}>
                  <SidebarMenuButton asChild isActive={active} className="app-nav-link"
                    tooltip={{ children: label, hidden: !compact }}>
                    <Link to={to} aria-label={label} aria-current={active ? (editor ? 'location' : 'page') : undefined} onClick={close}>
                      <Icon aria-hidden />
                      <span className="app-nav-label">{label}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              )
            })}
          </SidebarMenu>
        </nav>
      </SidebarContent>
      <SidebarFooter className="app-account">
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton className="profile-button" tooltip={{ children: name, hidden: !compact }} aria-label="Account menu">
                  <Avatar><AvatarFallback className="bg-secondary text-secondary-foreground text-xs">{initials}</AvatarFallback></Avatar>
                  <span className="app-account-name">{name}</span>
                  <ChevronUp className="app-account-chevron" aria-hidden />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" side={compact ? 'right' : 'top'} sideOffset={12} className="max-w-72 rounded-lg p-2">
                {user && <div className="truncate px-2 py-2 text-xs text-muted-foreground">{user.email}</div>}
                <DropdownMenuItem onSelect={() => logout()}><LogOut className="size-4" /> Log out</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </>
  )
}

function AppLayout({ editor, editorNavigationOpen, setEditorNavigationOpen }: {
  editor: boolean
  editorNavigationOpen: boolean
  setEditorNavigationOpen: (open: boolean) => void
}) {
  const { pathname } = useLocation()
  const { setOpenMobile, isMobile, triggerRef } = useSidebar()
  const main = useRef<HTMLElement | null>(null)

  useEffect(() => {
    setOpenMobile(false)
    setEditorNavigationOpen(false)
  }, [pathname, isMobile, setOpenMobile, setEditorNavigationOpen])

  function closeNavigation() {
    setOpenMobile(false)
    setEditorNavigationOpen(false)
  }

  return (
    <>
      <a className="app-skip-link" href="#app-main">Skip to content</a>
      <Sidebar collapsible="icon" className="app-sidebar" aria-label="Application sidebar">
        <AppSidebarContent close={closeNavigation} />
      </Sidebar>
      <main id="app-main" ref={main} tabIndex={-1} className="app-main">
        {!editor && (
          <header className="app-mobile-header">
            <SidebarTrigger className="app-navigation-trigger" />
            <Link to="/resumes" className="brand-link" aria-label="Resume Builder home"><Brand /></Link>
          </header>
        )}
        <Outlet />
      </main>
      <Sheet open={editor && !isMobile && editorNavigationOpen} onOpenChange={setEditorNavigationOpen}>
        <SheetContent side="left" showCloseButton={false} className="app-navigation-drawer"
          aria-describedby={undefined}
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            const button = triggerRef.current
            if (button?.isConnected && button.getClientRects().length) button.focus()
            else main.current?.focus()
          }}>
          <SheetTitle className="sr-only">Application navigation</SheetTitle>
          <AppSidebarContent overlay close={closeNavigation} />
        </SheetContent>
      </Sheet>
    </>
  )
}

export default function AppShell() {
  const { pathname } = useLocation()
  const editor = Boolean(matchPath('/resumes/:id/edit', pathname))
  const [collapsed, setCollapsed] = useState(() => {
    try { return localStorage.getItem(COLLAPSED_KEY) === 'true' } catch { return false }
  })
  const [editorNavigationOpen, setEditorNavigationOpen] = useState(false)

  function changeOpen(open: boolean) {
    if (editor) {
      setEditorNavigationOpen(open)
      return
    }
    setCollapsed(!open)
    try { localStorage.setItem(COLLAPSED_KEY, String(!open)) } catch { /* Keep navigation usable when storage is unavailable. */ }
  }

  return (
    <TooltipProvider delayDuration={200}>
      <SidebarProvider open={editor ? false : !collapsed} onOpenChange={changeOpen}
        style={{ '--sidebar-width': '15rem', '--sidebar-width-icon': '4.5rem' } as CSSProperties}
        className={cn('app-shell', editor && 'app-shell-editor', pathname === '/resumes' && 'app-shell-home')}>
        <AppLayout editor={editor} editorNavigationOpen={editorNavigationOpen} setEditorNavigationOpen={setEditorNavigationOpen} />
      </SidebarProvider>
    </TooltipProvider>
  )
}
