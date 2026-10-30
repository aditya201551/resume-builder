import { Navigate } from 'react-router'
import { ArrowRight, Check, GitBranch } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import { loginUrl } from '@/lib/api'
import { Button } from '@/components/ui/button'
import Brand from '@/components/layout/Brand'

export default function LoginPage() {
  const { user, isLoading } = useAuth()
  if (isLoading)
    return (
      <div
        role="status"
        className="flex min-h-svh items-center justify-center text-sm text-muted-foreground"
      >
        Opening your workspace…
      </div>
    )
  if (user) return <Navigate to="/resumes" replace />

  return (
    <main className="login-page">
      <header className="login-header">
        <Brand />
      </header>
      <div className="login-layout">
      <section className="login-intro">
        <h1>
          Your experience.
          <br />
          Your next chapter.
        </h1>
        <p className="login-description">
          Write your resume with AI, fine-tune every detail, and make it yours.
        </p>
        <div className="login-features">
          {[
            'AI writing partner',
            'Your layout, your way',
            'Ready to export',
          ].map((text) => (
            <span key={text} className="inline-flex items-center gap-1.5">
              <Check className="size-3.5 text-primary" />
              {text}
            </span>
          ))}
        </div>
      </section>
      <section className="login-signin" aria-labelledby="signin-heading">
        <h2 id="signin-heading">
          Make room for what’s next.
        </h2>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          Sign in to start your story, or pick up where you left off.
        </p>
        <div className="login-actions">
          <Button asChild className="h-11 w-full">
            <a href={loginUrl('google')}>
              Continue with Google <ArrowRight className="ml-auto size-4" />
            </a>
          </Button>
          <Button asChild variant="outline" className="h-11 w-full">
            <a href={loginUrl('github')}>
              <GitBranch className="size-4" />
              Continue with GitHub
            </a>
          </Button>
        </div>
        <p className="login-note">
          Your words and your experience.
          <br />
          You’re always in control.
        </p>
      </section>
      </div>
    </main>
  )
}
