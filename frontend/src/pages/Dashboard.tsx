import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { 
  Mail, 
  MessageSquare, 
  CheckCircle,
  Clock
} from 'lucide-react';

export function Dashboard() {
  // Mock data - would come from API
  const stats = {
    totalNotifications: 12453,
    successRate: 98.2,
    failureRate: 1.8,
    activeTemplates: 24,
    recentFailures: 12,
    pendingNotifications: 5,
  };

  const recentNotifications = [
    {
      id: '1',
      type: 'email',
      recipient: 'user@example.com',
      template: 'Welcome Email',
      status: 'sent',
      createdAt: '2026-03-15T08:30:00Z',
    },
    {
      id: '2', 
      type: 'sms',
      recipient: '+55 11 99999-9999',
      template: 'Order Confirmation',
      status: 'failed',
      createdAt: '2026-03-15T08:25:00Z',
      error: 'Invalid phone number'
    },
    // ... more notifications
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
        <p className="mt-2 text-gray-600">
          Overview of notification service performance and activity
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Total Notifications
            </CardTitle>
            <Mail className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.totalNotifications.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground">
              <span className="text-green-600">↗ +12%</span> from last month
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Success Rate
            </CardTitle>
            <CheckCircle className="h-4 w-4 text-green-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.successRate}%</div>
            <p className="text-xs text-muted-foreground">
              <span className="text-green-600">↗ +0.5%</span> from last week
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Active Templates
            </CardTitle>
            <MessageSquare className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.activeTemplates}</div>
            <p className="text-xs text-muted-foreground">
              3 created this week
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Pending
            </CardTitle>
            <Clock className="h-4 w-4 text-yellow-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.pendingNotifications}</div>
            <p className="text-xs text-muted-foreground">
              Awaiting processing
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Recent Activity */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Notifications</CardTitle>
          <CardDescription>
            Latest notification activity across all channels
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {recentNotifications.map((notification) => (
              <div key={notification.id} className="flex items-center justify-between p-4 border rounded-lg">
                <div className="flex items-center space-x-4">
                  <div className="flex-shrink-0">
                    {notification.type === 'email' ? (
                      <Mail className="h-5 w-5 text-blue-600" />
                    ) : (
                      <MessageSquare className="h-5 w-5 text-green-600" />
                    )}
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-900">
                      {notification.template}
                    </p>
                    <p className="text-sm text-gray-500">
                      To: {notification.recipient}
                    </p>
                    {notification.error && (
                      <p className="text-sm text-red-600">
                        Error: {notification.error}
                      </p>
                    )}
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    notification.status === 'sent'
                      ? 'bg-green-100 text-green-800'
                      : notification.status === 'failed'
                      ? 'bg-red-100 text-red-800'
                      : 'bg-yellow-100 text-yellow-800'
                  }`}>
                    {notification.status}
                  </span>
                  <span className="text-sm text-gray-500">
                    {new Date(notification.createdAt).toLocaleTimeString()}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}