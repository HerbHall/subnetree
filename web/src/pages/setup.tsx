import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export function SetupPage() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Welcome to NetVantage</CardTitle>
        <CardDescription>First-run setup wizard will be implemented in Issue #45.</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground">
          Create your admin account, configure your network, and run your first scan.
        </p>
      </CardContent>
    </Card>
  )
}
