import { Navigate } from 'react-router'
import { useAuth } from '@/hooks/useAuth'
import { loginUrl } from '@/lib/api'
import { Button } from '@/components/ui/button'

export default function LoginPage() {
  const { user, isLoading } = useAuth()

  if (isLoading) return null
  if (user) return <Navigate to="/resumes" replace />

  return (
    <div className="flex min-h-svh items-center justify-center bg-background px-6">
      <div className="flex w-full max-w-sm flex-col items-center gap-6 text-center">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">Resume Builder</h1>
          <p className="text-sm text-muted-foreground">Sign in to create and manage your resumes.</p>
        </div>
        <Button asChild className="w-full">
          <a href={loginUrl('google')}>Continue with Google</a>
        </Button>
      </div>
    </div>
  )
}
