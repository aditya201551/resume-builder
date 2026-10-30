import { Navigate } from 'react-router'
import { ArrowRight, Check, GitBranch, Sparkles } from 'lucide-react'
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
        Opening your workspace...
      </div>
    )
  if (user) return <Navigate to="/resumes" replace />

  return (
    <main className="grid min-h-svh items-center gap-12 bg-background p-6 lg:grid-cols-2 lg:p-16">
      <section className="mx-auto w-full max-w-lg">
        <Brand />
        <div className="mb-6 mt-16 inline-flex items-center gap-2 rounded-full bg-secondary px-3 py-1.5 text-xs text-primary">
          <Sparkles className="size-3.5" /> A little help. A lot of you.
        </div>
        <h1 className="text-4xl leading-tight font-semibold tracking-[-.05em] sm:text-5xl">
          Your experience.
          <br />
          <span className="text-primary">Your next chapter.</span>
        </h1>
        <p className="mt-6 max-w-sm text-sm leading-7 text-muted-foreground">
          A thoughtful space to build your resume. Write with AI, fine-tune
          every detail, and make it entirely yours.
        </p>
        <div className="mt-8 flex flex-wrap gap-x-5 gap-y-3 text-xs text-muted-foreground">
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
      <section className="mx-auto w-full max-w-md rounded-[28px] border border-border/60 bg-card p-8 shadow-[0_8px_36px_#2d2a3806] sm:p-10">
        <span className="mb-6 flex size-12 items-center justify-center rounded-2xl bg-secondary text-primary">
          <Sparkles className="size-6" />
        </span>
        <h2 className="text-2xl font-semibold tracking-tight">
          Make room for what’s next.
        </h2>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          Sign in to start your story, or pick up where you left off.
        </p>
        <div className="mt-8 grid gap-3">
          <Button asChild className="h-11 w-full rounded-xl">
            <a href={loginUrl('google')}>
              Continue with Google <ArrowRight className="ml-auto size-4" />
            </a>
          </Button>
          <Button asChild variant="outline" className="h-11 w-full rounded-xl">
            <a href={loginUrl('github')}>
              <GitBranch className="size-4" />
              Continue with GitHub
            </a>
          </Button>
        </div>
        <p className="mt-6 text-center text-xs leading-6 text-muted-foreground">
          Your words and your experience.
          <br />
          You're always in control.
        </p>
      </section>
    </main>
  )
}
